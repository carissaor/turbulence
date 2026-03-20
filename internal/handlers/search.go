package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
	"database/sql"

	mydb "github.com/carissaor/flight-tracker/internal/db"
	mw "github.com/carissaor/flight-tracker/internal/middleware"
	m "github.com/carissaor/flight-tracker/internal/models"
)

// GET /api/search
func HandleSearch(db *sql.DB, token string) http.HandlerFunc {
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

		routeID, _ := mydb.EnsureRoute(db, origin, destination)

		var results []m.SearchResult
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
					mydb.InsertPriceSnapshot(db, routeID, d.Price, &t)
				}
			}

			results = append(results, m.SearchResult{
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

		mw.WriteJSON(w, m.SearchResponse{
			Origin:      origin,
			Destination: destination,
			Month:       month,
			Results:     results,
		})
	}
}