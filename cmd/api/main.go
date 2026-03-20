package main

import (
	"database/sql"
	h "github.com/carissaor/flight-tracker/internal/handlers"
	mw "github.com/carissaor/flight-tracker/internal/middleware"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

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

	http.HandleFunc("/api/routes", mw.WithCORS(h.HandleRoutes(db)))
	http.HandleFunc("/api/prices", mw.WithCORS(h.HandlePrices(db)))
	http.HandleFunc("/api/events", mw.WithCORS(h.HandleEvents(db)))
	http.HandleFunc("/api/chaos", mw.WithCORS(h.HandleChaos(db)))
	http.HandleFunc("/api/search", mw.WithCORS(h.HandleSearch(db, token)))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 API server running", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
