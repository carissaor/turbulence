package db

import (
	"database/sql"
	"log"
	"time"
)

func EnsureRoute(db *sql.DB, origin, destination string) (int, error) {
	var id int
	err := db.QueryRow(`
		INSERT INTO routes (origin, destination)
		VALUES ($1, $2)
		ON CONFLICT (origin, destination) DO UPDATE SET origin = EXCLUDED.origin
		RETURNING id
	`, origin, destination).Scan(&id)

	return id, err
}

func InsertPriceSnapshot(db *sql.DB, routeID int, price float64, departDate *time.Time) {
	_, err := db.Exec(`
		INSERT INTO prices (route_id, price, currency, depart_date, fetched_at)
		VALUES ($1, $2, 'USD', $3, NOW())
	`, routeID, price, departDate)
	if err != nil {
		log.Println("Error inserting price snapshot:", err)
	}
}