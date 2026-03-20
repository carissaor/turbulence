package handlers

import (
	"database/sql"
	"net/http"
	"time"

	mw "github.com/carissaor/flight-tracker/internal/middleware"
	m "github.com/carissaor/flight-tracker/internal/models"
)

// GET /api/events
func HandleEvents(db *sql.DB) http.HandlerFunc {
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

		var events []m.EventResponse
		now := time.Now()

		for rows.Next() {
			var e m.EventResponse
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

		mw.WriteJSON(w, events)
	}
}