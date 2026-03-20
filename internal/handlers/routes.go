package handlers

import (
	"database/sql"
	"net/http"

	m "github.com/carissaor/flight-tracker/internal/models"
	mw "github.com/carissaor/flight-tracker/internal/middleware"
)

// GET /api/routes
func HandleRoutes(db *sql.DB) http.HandlerFunc {
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

		var routes []m.RouteResponse

		for rows.Next() {
			var rt m.RouteResponse
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

		mw.WriteJSON(w, routes)
	}
}