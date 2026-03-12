package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Flight struct {
	Departure struct {
		Airport   string `json:"airport"`
		Scheduled string `json:"scheduled"`
	} `json:"departure"`
	Arrival struct {
		Airport string `json:"airport"`
	} `json:"arrival"`
	Airline struct {
		Name string `json:"name"`
	} `json:"airline"`
	Flight struct {
		Iata string `json:"iata"`
	} `json:"flight"`
}

type Response struct {
	Data []Flight `json:"data"`
}

func fetchFlights(apiKey string) ([]Flight, error) {
	url := fmt.Sprintf("http://api.aviationstack.com/v1/flights?access_key=%s&dep_iata=YVR&limit=5", apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result Response
	json.Unmarshal(body, &result)
	return result.Data, nil
}

func saveFlights(db *sql.DB, flights []Flight) {
	for _, f := range flights {
		scheduledTime, err := time.Parse(time.RFC3339, f.Departure.Scheduled)
		if err != nil {
			log.Println("Error parsing time:", err)
			continue
		}

		_, err = db.Exec(`
			INSERT INTO flights (flight_iata, departure_airport, arrival_airport, scheduled_departure)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT DO NOTHING`,
			f.Flight.Iata,
			f.Departure.Airport,
			f.Arrival.Airport,
			scheduledTime,
		)
		if err != nil {
			log.Println("Error inserting flight:", err)
		} else {
			fmt.Printf("✅ Saved: %s | %s → %s\n", f.Flight.Iata, f.Departure.Airport, f.Arrival.Airport)
		}
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Cannot reach database:", err)
	}
	fmt.Println("🐘 Connected to PostgreSQL!")

	flights, err := fetchFlights(os.Getenv("AVIATIONSTACK_API_KEY"))
	if err != nil {
		log.Fatal("Error fetching flights:", err)
	}

	saveFlights(db, flights)
}