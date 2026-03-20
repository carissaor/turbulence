package models

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
	Date  string  `json:"date"`
	Price float64 `json:"price"`
}

type EventResponse struct {
	Question    string  `json:"question"`
	Probability float64 `json:"probability"`
	Volume      float64 `json:"volume"`
	EndDate     string  `json:"end_date"`
	FetchedAt   string  `json:"fetched_at"`
}

type ChaosResponse struct {
	Score       float64 `json:"score"`
	Level       string  `json:"level"`
	Label       string  `json:"label"`
	Insight     string  `json:"insight"`
	MarketCount int     `json:"market_count"`
}

type SearchResult struct {
	Origin      string  `json:"origin"`
	Destination string  `json:"destination"`
	Price       float64 `json:"price"`
	DepartDate  string  `json:"depart_date"`
	Airline     string  `json:"airline"`
	Transfers   int     `json:"transfers"`
}

type SearchResponse struct {
	Origin      string         `json:"origin"`
	Destination string         `json:"destination"`
	Month       string         `json:"month"`
	Results     []SearchResult `json:"results"`
}

type PriceEntry struct {
	Price      float64 `json:"price"`
	DepartDate string  `json:"departure_at"`
	Airline    string  `json:"airline"`
	FlightNum  int     `json:"flight_number"`
	Transfers  int     `json:"transfers"`
}

type PriceResponse struct {
	Success bool                                `json:"success"`
	Data    map[string]map[string]PriceEntry `json:"data"`
}