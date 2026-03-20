package handlers

import (
	"database/sql"
	"net/http"

	mw "github.com/carissaor/flight-tracker/internal/middleware"
	m "github.com/carissaor/flight-tracker/internal/models"
)

// GET /api/prices?route=YVR-LHR&mode=depart
// mode=depart       -> latest price by departure date
// mode=dailyLowest  -> lowest observed price by fetched day
func HandlePrices(db *sql.DB) http.HandlerFunc {
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

		var points []m.PricePoint
		for rows.Next() {
			var pt m.PricePoint
			var dt sql.NullTime

			if err := rows.Scan(&dt, &pt.Price); err != nil {
				continue
			}

			if dt.Valid {
				pt.Date = dt.Time.Format("2006-01-02")
				points = append(points, pt)
			}
		}

		mw.WriteJSON(w, m.PriceHistoryResponse{
			Route:  route,
			Prices: points,
		})
	}
}