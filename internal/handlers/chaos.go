package handlers

import (
	"database/sql"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	mw "github.com/carissaor/flight-tracker/internal/middleware"
	m "github.com/carissaor/flight-tracker/internal/models"
)

// GET /api/chaos
func HandleChaos(db *sql.DB) http.HandlerFunc {
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
			mw.WriteJSON(w, m.ChaosResponse{
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

		mw.WriteJSON(w, m.ChaosResponse{
			Score:       math.Round(score*10) / 10,
			Level:       level,
			Label:       label,
			Insight:     insight,
			MarketCount: count,
		})
	}
}

func adjustedSignal(question string, probability float64) (float64, float64) {
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