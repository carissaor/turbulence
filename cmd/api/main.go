package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// ---------------------------------------------------------------------------
// API response types
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
	Price      float64 `json:"price"`
	DepartDate string  `json:"depart_date"`
	FetchedAt  string  `json:"fetched_at"`
}

type EventResponse struct {
	Question    string  `json:"question"`
	Probability float64 `json:"probability"`
	Volume      float64 `json:"volume"`
	FetchedAt   string  `json:"fetched_at"`
}

type ChaosResponse struct {
	Score       float64 `json:"score"`
	Level       string  `json:"level"`
	Label       string  `json:"label"`
	Insight     string  `json:"insight"`
	MarketCount int     `json:"market_count"`
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
// Handlers
// ---------------------------------------------------------------------------

// GET /api/routes
// Returns all tracked routes with their latest and lowest price
func handleRoutes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT
				r.id,
				r.origin,
				r.destination,
				MIN(p.price) AS lowest_price,
				(SELECT price FROM prices WHERE route_id = r.id ORDER BY fetched_at DESC LIMIT 1) AS latest_price,
				(SELECT depart_date FROM prices WHERE route_id = r.id ORDER BY fetched_at DESC LIMIT 1) AS depart_date
			FROM routes r
			LEFT JOIN prices p ON p.route_id = r.id
			GROUP BY r.id, r.origin, r.destination
			ORDER BY r.destination
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
			if err := rows.Scan(&rt.ID, &rt.Origin, &rt.Destination, &rt.LowestPrice, &rt.LatestPrice, &departDate); err != nil {
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

// GET /api/prices?route=YVR-LHR
// Returns price history for a specific route
func handlePrices(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		route := r.URL.Query().Get("route") // e.g. "YVR-LHR"
		if route == "" {
			http.Error(w, "missing ?route= parameter (e.g. YVR-LHR)", http.StatusBadRequest)
			return
		}

		// Split "YVR-LHR" into origin and destination
		if len(route) != 7 || route[3] != '-' {
			http.Error(w, "route must be in format YVR-LHR", http.StatusBadRequest)
			return
		}
		origin := route[:3]
		dest := route[4:]

		rows, err := db.Query(`
			SELECT p.price, p.depart_date, p.fetched_at
			FROM prices p
			JOIN routes r ON r.id = p.route_id
			WHERE r.origin = $1 AND r.destination = $2
			ORDER BY p.fetched_at ASC
		`, origin, dest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var points []PricePoint
		for rows.Next() {
			var pt PricePoint
			var departDate sql.NullTime
			var fetchedAt sql.NullTime
			if err := rows.Scan(&pt.Price, &departDate, &fetchedAt); err != nil {
				continue
			}
			if departDate.Valid {
				pt.DepartDate = departDate.Time.Format("2006-01-02")
			}
			if fetchedAt.Valid {
				pt.FetchedAt = fetchedAt.Time.Format("2006-01-02T15:04:05Z")
			}
			points = append(points, pt)
		}

		writeJSON(w, PriceHistoryResponse{
			Route:  route,
			Prices: points,
		})
	}
}

// GET /api/events
// Returns the latest world event signals from Polymarket
func handleEvents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT DISTINCT ON (question)
				question, probability, volume, fetched_at
			FROM events
			ORDER BY question, fetched_at DESC
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var events []EventResponse
		for rows.Next() {
			var e EventResponse
			var fetchedAt sql.NullTime
			if err := rows.Scan(&e.Question, &e.Probability, &e.Volume, &fetchedAt); err != nil {
				continue
			}
			if fetchedAt.Valid {
				e.FetchedAt = fetchedAt.Time.Format("2006-01-02T15:04:05Z")
			}
			events = append(events, e)
		}
		writeJSON(w, events)
	}
}

func adjustedSignal(question string, probability float64) (signal float64, weight float64) {
	q := strings.ToLower(question)
 
	// Peace/ceasefire markets — invert probability
	// Low ceasefire probability = high conflict = high chaos
	if strings.Contains(q, "ceasefire") || strings.Contains(q, "peace deal") || strings.Contains(q, "peace agreement") {
		return 1 - probability, 2.0
	}
 
	// War/conflict/invasion markets — high weight, use probability directly
	if strings.Contains(q, "declare war") || strings.Contains(q, "invasion") || strings.Contains(q, "invade") || strings.Contains(q, "attack") {
		return probability, 3.0
	}
 
	// Pandemic/health emergency markets — high weight
	if strings.Contains(q, "pandemic") || strings.Contains(q, "health emergency") || strings.Contains(q, "who declares") {
		return probability, 2.5
	}
 
	// Travel ban / airspace markets — high weight, direct impact on flights
	if strings.Contains(q, "travel ban") || strings.Contains(q, "airspace") {
		return probability, 3.0
	}
 
	// Oil markets — weight by price threshold
	// Only extreme price spikes meaningfully affect flight prices
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
 
	// Default — use probability with neutral weight
	return probability, 1.0
}
 
// extractOilThreshold tries to parse the price threshold from an oil market question
// e.g. "Will Crude Oil hit $150 by end of March?" → 150
func extractOilThreshold(q string) float64 {
	// Find dollar sign and parse number after it
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
 
// GET /api/chaos
// Returns a weighted chaos score based on adjusted Polymarket signals
func handleChaos(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT DISTINCT ON (question)
				question, probability, volume
			FROM events
			ORDER BY question, fetched_at DESC
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
 
		var totalWeight float64
		var weightedSum float64
		var count int
 
		for rows.Next() {
			var question string
			var prob, volume float64
			if err := rows.Scan(&question, &prob, &volume); err != nil {
				continue
			}
 
			signal, typeWeight := adjustedSignal(question, prob)
 
			// Volume weight — more reliable signals get more say
			volumeWeight := (volume/100000 + 1) * typeWeight
 
			weightedSum += signal * volumeWeight
			totalWeight += volumeWeight
			count++
		}
 
		if count == 0 || totalWeight == 0 {
			writeJSON(w, ChaosResponse{
				Score:       0,
				Level:       "UNKNOWN",
				Label:       "no idea tbh 🤷",
				Insight:     "Run the collector to start tracking world events.",
				MarketCount: 0,
			})
			return
		}
 
		score := (weightedSum / totalWeight) * 100
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

	http.HandleFunc("/api/routes", withCORS(handleRoutes(db)))
	http.HandleFunc("/api/prices", withCORS(handlePrices(db)))
	http.HandleFunc("/api/events", withCORS(handleEvents(db)))
	http.HandleFunc("/api/chaos", withCORS(handleChaos(db)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 API server running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
