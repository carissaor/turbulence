package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	mydb "github.com/carissaor/flight-tracker/internal/db"
	mw "github.com/carissaor/flight-tracker/internal/middleware"
	m "github.com/carissaor/flight-tracker/internal/models"
)

const (
	travelPayoutsCalendarURL = "https://api.travelpayouts.com/v1/prices/calendar"

	// Protect the server from unexpectedly large upstream responses.
	maxUpstreamBodyBytes = 2 << 20 // 2 MB
)

// Reuse one HTTP client so Go can reuse connections efficiently.
var travelPayoutsClient = &http.Client{
	Timeout: 10 * time.Second,
}

// GET /api/search?origin=YVR&destination=LHR&month=2026-10
func HandleSearch(db *sql.DB, token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(
				w,
				"method not allowed",
				http.StatusMethodNotAllowed,
			)
			return
		}

		origin := strings.ToUpper(
			strings.TrimSpace(
				r.URL.Query().Get("origin"),
			),
		)

		destination := strings.ToUpper(
			strings.TrimSpace(
				r.URL.Query().Get("destination"),
			),
		)

		month := strings.TrimSpace(
			r.URL.Query().Get("month"),
		)

		if origin == "" || destination == "" || month == "" {
			http.Error(
				w,
				"missing origin, destination, or month",
				http.StatusBadRequest,
			)
			return
		}

		/*
		 * Validate airport codes.
		 */
		if !isAirportCode(origin) || !isAirportCode(destination) {
			http.Error(
				w,
				"origin and destination must be 3-letter airport codes",
				http.StatusBadRequest,
			)
			return
		}

		if origin == destination {
			http.Error(
				w,
				"origin and destination must be different",
				http.StatusBadRequest,
			)
			return
		}

		/*
		 * Validate month format.
		 *
		 * Expected:
		 * 2026-10
		 */
		if _, err := time.Parse("2006-01", month); err != nil {
			http.Error(
				w,
				"month must be in YYYY-MM format",
				http.StatusBadRequest,
			)
			return
		}

		/*
		 * The token is application configuration rather
		 * than a user input problem.
		 */
		if strings.TrimSpace(token) == "" {
			log.Printf(
				"search failed: Travelpayouts token is not configured",
			)

			http.Error(
				w,
				"flight price service is not configured",
				http.StatusInternalServerError,
			)
			return
		}

		/*
		 * ------------------------------------------------------------
		 * BUILD TRAVELPAYOUTS REQUEST
		 * ------------------------------------------------------------
		 *
		 * Use url.Values instead of manually constructing
		 * the query string.
		 */
		params := url.Values{}

		params.Set("origin", origin)
		params.Set("destination", destination)
		params.Set("depart_date", month)
		params.Set("calendar_type", "departure_date")
		params.Set("currency", "usd")
		params.Set("token", token)

		requestURL :=
			travelPayoutsCalendarURL +
				"?" +
				params.Encode()

		/*
		 * Use the incoming request context.
		 *
		 * If the browser disconnects/cancels, the upstream
		 * request can also be cancelled.
		 */
		req, err := http.NewRequestWithContext(
			r.Context(),
			http.MethodGet,
			requestURL,
			nil,
		)

		if err != nil {
			log.Printf(
				"failed to create Travelpayouts request for %s-%s: %v",
				origin,
				destination,
				err,
			)

			http.Error(
				w,
				"could not create flight search request",
				http.StatusInternalServerError,
			)
			return
		}

		/*
		 * Keep the existing token header as well.
		 */
		req.Header.Set(
			"X-Access-Token",
			token,
		)

		/*
		 * ------------------------------------------------------------
		 * CALL TRAVELPAYOUTS
		 * ------------------------------------------------------------
		 */
		resp, err := travelPayoutsClient.Do(req)

		if err != nil {
			log.Printf(
				"Travelpayouts request failed for %s-%s %s: %v",
				origin,
				destination,
				month,
				err,
			)

			http.Error(
				w,
				"flight price service unavailable",
				http.StatusBadGateway,
			)
			return
		}

		defer resp.Body.Close()

		/*
		 * Check the HTTP status BEFORE attempting
		 * to parse the response.
		 */
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			log.Printf(
				"Travelpayouts returned HTTP %d for %s-%s %s",
				resp.StatusCode,
				origin,
				destination,
				month,
			)

			http.Error(
				w,
				"flight price service returned an error",
				http.StatusBadGateway,
			)
			return
		}

		/*
		 * ------------------------------------------------------------
		 * PARSE RESPONSE
		 * ------------------------------------------------------------
		 */
		var calResp struct {
			Success bool `json:"success"`

			Data map[string]struct {
				Origin      string  `json:"origin"`
				Destination string  `json:"destination"`
				Price       float64 `json:"price"`
				Transfers   int     `json:"transfers"`
				Airline     string  `json:"airline"`
				DepartureAt string  `json:"departure_at"`
			} `json:"data"`
		}

		decoder := json.NewDecoder(
			io.LimitReader(
				resp.Body,
				maxUpstreamBodyBytes,
			),
		)

		if err := decoder.Decode(&calResp); err != nil {
			log.Printf(
				"failed to decode Travelpayouts response for %s-%s: %v",
				origin,
				destination,
				err,
			)

			http.Error(
				w,
				"invalid response from flight price service",
				http.StatusBadGateway,
			)
			return
		}

		if !calResp.Success {
			log.Printf(
				"Travelpayouts reported unsuccessful search for %s-%s %s",
				origin,
				destination,
				month,
			)

			http.Error(
				w,
				"flight price service could not complete the search",
				http.StatusBadGateway,
			)
			return
		}

		/*
		 * ------------------------------------------------------------
		 * ENSURE ROUTE EXISTS
		 * ------------------------------------------------------------
		 *
		 * Price history depends on this route existing,
		 * so don't silently ignore a DB failure.
		 */
		routeID, err := mydb.EnsureRoute(
			db,
			origin,
			destination,
		)

		if err != nil {
			log.Printf(
				"failed to ensure route %s-%s: %v",
				origin,
				destination,
				err,
			)

			http.Error(
				w,
				"could not save flight route",
				http.StatusInternalServerError,
			)
			return
		}

		if routeID <= 0 {
			log.Printf(
				"EnsureRoute returned invalid route ID for %s-%s",
				origin,
				destination,
			)

			http.Error(
				w,
				"could not save flight route",
				http.StatusInternalServerError,
			)
			return
		}

		/*
		 * ------------------------------------------------------------
		 * PROCESS RESULTS
		 * ------------------------------------------------------------
		 *
		 * Start with an empty non-nil slice so an empty
		 * result is returned as:
		 *
		 * []
		 *
		 * rather than:
		 *
		 * null
		 */
		results := make(
			[]m.SearchResult,
			0,
			len(calResp.Data),
		)

		snapshotFailures := 0

		for _, d := range calResp.Data {
			/*
			 * Ignore invalid prices.
			 */
			if d.Price <= 0 {
				continue
			}

			/*
			 * Travelpayouts departure_at should contain
			 * at least YYYY-MM-DD.
			 */
			if len(d.DepartureAt) < 10 {
				log.Printf(
					"skipping result with invalid departure_at %q for %s-%s",
					d.DepartureAt,
					origin,
					destination,
				)

				continue
			}

			departDate := d.DepartureAt[:10]

			/*
			 * Validate the date instead of trusting the
			 * first ten characters.
			 */
			parsedDepartDate, err := time.Parse(
				"2006-01-02",
				departDate,
			)

			if err != nil {
				log.Printf(
					"skipping invalid departure date %q for %s-%s: %v",
					departDate,
					origin,
					destination,
					err,
				)

				continue
			}

			/*
			 * Only include results belonging to the
			 * requested month.
			 */
			if parsedDepartDate.Format("2006-01") != month {
				continue
			}

			/*
			 * --------------------------------------------------------
			 * SAVE PRICE SNAPSHOT
			 * --------------------------------------------------------
			 *
			 * A single failed snapshot should not destroy
			 * an otherwise successful flight search.
			 *
			 * Log it, count it, and continue returning
			 * the valid search results.
			 */
			if err := mydb.InsertPriceSnapshot(
				db,
				routeID,
				d.Price,
				&parsedDepartDate,
			); err != nil {
				snapshotFailures++

				log.Printf(
					"failed to save price snapshot for %s-%s departing %s: %v",
					origin,
					destination,
					departDate,
					err,
				)
			}

			/*
			 * Add the flight to the API response.
			 */
			results = append(
				results,
				m.SearchResult{
					Origin:      origin,
					Destination: destination,
					Price:       d.Price,
					DepartDate:  departDate,
					Airline:     d.Airline,
					Transfers:   d.Transfers,
				},
			)
		}

		if snapshotFailures > 0 {
			log.Printf(
				"search %s-%s %s completed with %d snapshot persistence failure(s)",
				origin,
				destination,
				month,
				snapshotFailures,
			)
		}

		/*
		 * ------------------------------------------------------------
		 * SORT RESULTS
		 * ------------------------------------------------------------
		 *
		 * Cheapest flights first.
		 *
		 * If prices are equal, show the earlier
		 * departure first for deterministic output.
		 */
		sort.SliceStable(
			results,
			func(i, j int) bool {
				if results[i].Price == results[j].Price {
					return results[i].DepartDate <
						results[j].DepartDate
				}

				return results[i].Price <
					results[j].Price
			},
		)

		/*
		 * ------------------------------------------------------------
		 * RESPONSE
		 * ------------------------------------------------------------
		 */
		mw.WriteJSON(
			w,
			m.SearchResponse{
				Origin:      origin,
				Destination: destination,
				Month:       month,
				Results:     results,
			},
		)
	}
}

/*
 * isAirportCode performs lightweight validation
 * for the airport codes accepted by this endpoint.
 *
 * Examples:
 *
 * YVR -> true
 * LHR -> true
 * YV1 -> false
 * YYVR -> false
 */
func isAirportCode(code string) bool {
	if len(code) != 3 {
		return false
	}

	for _, ch := range code {
		if ch < 'A' || ch > 'Z' {
			return false
		}
	}

	return true
}
