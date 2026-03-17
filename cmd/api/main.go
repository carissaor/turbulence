package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type RouteResponse struct {
	ID          int     `json:"id"`
	Origin      string  `json:"origin"`
	Destination string  `json:"destination"`
	LowestPrice float64 `json:"lowest_price"`
	LatestPrice float64 `json:"latest_price"`
	DepartDate  string  `json:"depart_date"`
}

type PriceHistoryResponse struct {
	Route  string       `json:"route"`
	Prices []PricePoint `json:"prices"`
}

type PricePoint struct {
	Date  string  `json:"date"`
	Price float64 `json:"price"`
}

type EventResponse struct {
	Question    string  `json:"question"`
	Probability float64 `json:"probability"`
	Volume      float64 `json:"volume"`
	EndDate     string  `json:"end_date"`
	FetchedAt   string  `json:"fetched_at"`
}

type ChaosResponse struct {
	Score       float64 `json:"score"`
	Level       string  `json:"level"`
	Label       string  `json:"label"`
	Insight     string  `json:"insight"`
	MarketCount int     `json:"market_count"`
}

type SearchResult struct {
	Origin      string  `json:"origin"`
	Destination string  `json:"destination"`
	Price       float64 `json:"price"`
	DepartDate  string  `json:"depart_date"`
	Airline     string  `json:"airline"`
	Transfers   int     `json:"transfers"`
}

type SearchResponse struct {
	Origin      string         `json:"origin"`
	Destination string         `json:"destination"`
	Month       string         `json:"month"`
	Results     []SearchResult `json:"results"`
}

// Shared with collector
type PriceEntry struct {
	Price      float64 `json:"price"`
	DepartDate string  `json:"departure_at"`
	Airline    string  `json:"airline"`
	FlightNum  int     `json:"flight_number"`
	Transfers  int     `json:"transfers"`
}

type PriceResponse struct {
	Success bool                             `json:"success"`
	Data    map[string]map[string]PriceEntry `json:"data"`
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

func withCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, data any) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// ---------------------------------------------------------------------------
// Shared DB helpers
// ---------------------------------------------------------------------------

func ensureRoute(db *sql.DB, origin, destination string) (int, error) {
	var id int
	err := db.QueryRow(`
		INSERT INTO routes (origin, destination)
		VALUES ($1, $2)
		ON CONFLICT (origin, destination) DO UPDATE SET origin = EXCLUDED.origin
		RETURNING id
	`, origin, destination).Scan(&id)

	return id, err
}

func insertPriceSnapshot(db *sql.DB, routeID int, price float64, departDate *time.Time) {
	_, err := db.Exec(`
		INSERT INTO prices (route_id, price, currency, depart_date, fetched_at)
		VALUES ($1, $2, 'USD', $3, NOW())
	`, routeID, price, departDate)
	if err != nil {
		log.Println("Error inserting price snapshot:", err)
	}
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// GET /api/routes
func handleRoutes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT
				r.id,
				r.origin,
				r.destination,
				COALESCE(MIN(p.price), 0) AS lowest_price,
				COALESCE(lp.price, 0) AS latest_price,
				lp.depart_date
			FROM routes r
			LEFT JOIN prices p ON p.route_id = r.id
			LEFT JOIN LATERAL (
				SELECT price, depart_date
				FROM prices
				WHERE route_id = r.id
				ORDER BY fetched_at DESC
				LIMIT 1
			) lp ON true
			GROUP BY r.id, r.origin, r.destination, lp.price, lp.depart_date
			ORDER BY latest_price ASC
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var routes []RouteResponse

		for rows.Next() {
			var rt RouteResponse
			var departDate sql.NullTime

			if err := rows.Scan(
				&rt.ID,
				&rt.Origin,
				&rt.Destination,
				&rt.LowestPrice,
				&rt.LatestPrice,
				&departDate,
			); err != nil {
				continue
			}

			if departDate.Valid {
				rt.DepartDate = departDate.Time.Format("2006-01-02")
			}

			routes = append(routes, rt)
		}

		writeJSON(w, routes)
	}
}

// GET /api/prices?route=YVR-LHR&mode=depart
// mode=depart       -> latest price by departure date
// mode=dailyLowest  -> lowest observed price by fetched day
func handlePrices(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		route := r.URL.Query().Get("route")
		mode := r.URL.Query().Get("mode")
		if mode == "" {
			mode = "depart"
		}

		if route == "" {
			http.Error(w, "missing ?route= parameter (e.g. YVR-LHR)", http.StatusBadRequest)
			return
		}
		if len(route) != 7 || route[3] != '-' {
			http.Error(w, "route must be in format YVR-LHR", http.StatusBadRequest)
			return
		}

		origin := route[:3]
		dest := route[4:]

		var (
			rows *sql.Rows
			err  error
		)

		switch mode {
		case "dailyLowest":
			rows, err = db.Query(`
				SELECT
					DATE(p.fetched_at) AS date,
					MIN(p.price) AS price
				FROM prices p
				JOIN routes r ON r.id = p.route_id
				WHERE r.origin = $1
					AND r.destination = $2
				GROUP BY DATE(p.fetched_at)
				ORDER BY date ASC
			`, origin, dest)

		case "depart":
			fallthrough
		default:
			rows, err = db.Query(`
				SELECT DISTINCT ON (p.depart_date)
					p.depart_date AS date,
					p.price
				FROM prices p
				JOIN routes r ON r.id = p.route_id
				WHERE r.origin = $1
					AND r.destination = $2
					AND p.depart_date IS NOT NULL
				ORDER BY p.depart_date, p.fetched_at DESC
			`, origin, dest)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var points []PricePoint
		for rows.Next() {
			var pt PricePoint
			var dt sql.NullTime

			if err := rows.Scan(&dt, &pt.Price); err != nil {
				continue
			}

			if dt.Valid {
				pt.Date = dt.Time.Format("2006-01-02")
				points = append(points, pt)
			}
		}

		writeJSON(w, PriceHistoryResponse{
			Route:  route,
			Prices: points,
		})
	}
}

// GET /api/events
func handleEvents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT DISTINCT ON (question)
				question,
				probability,
				volume,
				end_date,
				fetched_at
			FROM events
			ORDER BY question, fetched_at DESC
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var events []EventResponse
		now := time.Now()

		for rows.Next() {
			var e EventResponse
			var endDate sql.NullTime
			var fetchedAt sql.NullTime

			if err := rows.Scan(&e.Question, &e.Probability, &e.Volume, &endDate, &fetchedAt); err != nil {
				continue
			}

			if endDate.Valid && endDate.Time.Before(now) {
				continue
			}

			if e.Probability <= 0.01 || e.Probability >= 0.99 {
				continue
			}

			if endDate.Valid {
				e.EndDate = endDate.Time.Format(time.RFC3339)
			}
			if fetchedAt.Valid {
				e.FetchedAt = fetchedAt.Time.Format(time.RFC3339)
			}

			events = append(events, e)
		}

		writeJSON(w, events)
	}
}

// GET /api/chaos
func handleChaos(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT DISTINCT ON (question)
				question,
				probability,
				volume,
				end_date
			FROM events
			ORDER BY question, fetched_at DESC
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var weightedSum float64
		var totalWeight float64
		var count int
		now := time.Now()

		for rows.Next() {
			var question string
			var prob float64
			var volume float64
			var endDate sql.NullTime

			if err := rows.Scan(&question, &prob, &volume, &endDate); err != nil {
				continue
			}

			if endDate.Valid && endDate.Time.Before(now) {
				continue
			}

			if prob <= 0.01 || prob >= 0.99 {
				continue
			}

			signal, typeWeight := adjustedSignal(question, prob)
			volumeWeight := math.Log10(volume+100) * typeWeight
			uncertainty := 1 - math.Abs(prob-0.5)*2

			timeWeight := 1.0
			if endDate.Valid {
				days := endDate.Time.Sub(now).Hours() / 24
				switch {
				case days < 7:
					timeWeight = 2.0
				case days < 30:
					timeWeight = 1.5
				case days < 90:
					timeWeight = 1.2
				}
			}

			eventWeight := volumeWeight * uncertainty * timeWeight
			weightedSum += signal * eventWeight
			totalWeight += eventWeight
			count++
		}

		if totalWeight == 0 {
			writeJSON(w, ChaosResponse{
				Score:       0,
				Level:       "UNKNOWN",
				Label:       "no idea tbh 🤷",
				Insight:     "Run the collector to start tracking events.",
				MarketCount: 0,
			})
			return
		}

		score := math.Min((weightedSum/totalWeight)*120, 100)
		level, label, insight := chaosLevel(score)

		writeJSON(w, ChaosResponse{
			Score:       math.Round(score*10) / 10,
			Level:       level,
			Label:       label,
			Insight:     insight,
			MarketCount: count,
		})
	}
}

// GET /api/search
func handleSearch(db *sql.DB, token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := strings.ToUpper(r.URL.Query().Get("origin"))
		destination := strings.ToUpper(r.URL.Query().Get("destination"))
		month := r.URL.Query().Get("month")

		if origin == "" || destination == "" || month == "" {
			http.Error(w, "missing origin, destination, or month", http.StatusBadRequest)
			return
		}

		url := fmt.Sprintf(
			"http://api.travelpayouts.com/v1/prices/calendar?origin=%s&destination=%s&depart_date=%s&calendar_type=departure_date&currency=usd&token=%s",
			origin, destination, month, token,
		)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req.Header.Set("X-Access-Token", token)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var calResp struct {
			Success bool `json:"success"`
			Data    map[string]struct {
				Origin      string  `json:"origin"`
				Destination string  `json:"destination"`
				Price       float64 `json:"price"`
				Transfers   int     `json:"transfers"`
				Airline     string  `json:"airline"`
				DepartureAt string  `json:"departure_at"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &calResp); err != nil {
			http.Error(w, "failed to parse response", http.StatusInternalServerError)
			return
		}

		if !calResp.Success {
			http.Error(w, "Travelpayouts returned error", http.StatusBadGateway)
			return
		}

		routeID, _ := ensureRoute(db, origin, destination)

		var results []SearchResult
		for _, d := range calResp.Data {
			if d.Price == 0 {
				continue
			}

			departDate := ""
			if len(d.DepartureAt) >= 10 {
				departDate = d.DepartureAt[:10]
			}

			if !strings.HasPrefix(departDate, month) {
				continue
			}

			if routeID > 0 && departDate != "" {
				t, err := time.Parse("2006-01-02", departDate)
				if err == nil {
					insertPriceSnapshot(db, routeID, d.Price, &t)
				}
			}

			results = append(results, SearchResult{
				Origin:      origin,
				Destination: destination,
				Price:       d.Price,
				DepartDate:  departDate,
				Airline:     d.Airline,
				Transfers:   d.Transfers,
			})
		}

		sort.Slice(results, func(i, j int) bool {
			return results[i].Price < results[j].Price
		})

		writeJSON(w, SearchResponse{
			Origin:      origin,
			Destination: destination,
			Month:       month,
			Results:     results,
		})
	}
}

// ---------------------------------------------------------------------------
// Chaos helpers
// ---------------------------------------------------------------------------

func adjustedSignal(question string, probability float64) (signal float64, weight float64) {
	q := strings.ToLower(question)

	if strings.Contains(q, "ceasefire") || strings.Contains(q, "peace deal") || strings.Contains(q, "peace agreement") {
		return 1 - probability, 2.0
	}
	if strings.Contains(q, "declare war") || strings.Contains(q, "invasion") || strings.Contains(q, "invade") || strings.Contains(q, "attack") {
		return probability, 3.0
	}
	if strings.Contains(q, "pandemic") || strings.Contains(q, "health emergency") || strings.Contains(q, "who declares") {
		return probability, 2.5
	}
	if strings.Contains(q, "travel ban") || strings.Contains(q, "airspace") {
		return probability, 3.0
	}
	if strings.Contains(q, "crude oil") || strings.Contains(q, " oil ") {
		threshold := extractOilThreshold(q)
		switch {
		case threshold >= 200:
			return probability, 3.0
		case threshold >= 150:
			return probability, 2.0
		case threshold >= 120:
			return probability, 1.0
		default:
			return probability, 0.2
		}
	}

	return probability, 1.0
}

func extractOilThreshold(q string) float64 {
	idx := strings.Index(q, "$")
	if idx == -1 {
		return 0
	}

	numStr := ""
	for _, c := range q[idx+1:] {
		if c >= '0' && c <= '9' {
			numStr += string(c)
		} else if c == ',' {
			continue
		} else {
			break
		}
	}

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	return val
}

func chaosLevel(score float64) (string, string, string) {
	switch {
	case score >= 60:
		return "EXTREME", "We are so cooked 😭", "Book ASAP and get a refundable ticket!"
	case score >= 40:
		return "HIGH", "It's giving chaos 🌪️", "Things are getting spicy...don't wait!"
	case score >= 20:
		return "MODERATE", "sus but manageable 👀", "Could be nothing. Could be everything. Check back soon!"
	default:
		return "LOW", "Calm Skies ✌️", "Weirdly calm, book before that changes!"
	}
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	godotenv.Load()

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Cannot reach database:", err)
	}
	log.Println("🐘 Connected to PostgreSQL!")

	token := os.Getenv("TRAVELPAYOUTS_TOKEN")

	http.HandleFunc("/api/routes", withCORS(handleRoutes(db)))
	http.HandleFunc("/api/prices", withCORS(handlePrices(db)))
	http.HandleFunc("/api/events", withCORS(handleEvents(db)))
	http.HandleFunc("/api/chaos", withCORS(handleChaos(db)))
	http.HandleFunc("/api/search", withCORS(handleSearch(db, token)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 API server running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}