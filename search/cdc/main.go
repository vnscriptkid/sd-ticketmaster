package main

import (
	"context"
	"fmt"
	"log"

	"github.com/olivere/elastic/v7"
)

type Venue struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	City    string `json:"city"`
	State   string `json:"state"`
	Country string `json:"country"`
}

type Event struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Venue     Venue  `json:"venue"`
	EventDate string `json:"event_date"`
}

func main() {
	client, err := elastic.NewClient(
		elastic.SetURL("http://localhost:9200"),
		elastic.SetSniff(false),
	)
	if err != nil {
		log.Fatalf("Error creating the Elasticsearch client: %s", err)
	}

	events := []Event{
		{ID: "1", Name: "Concert A", Venue: Venue{"v1", "Stadium A", "New York", "NY", "USA"}, EventDate: "2024-12-01"},
		{ID: "2", Name: "Concert B", Venue: Venue{"v2", "Stadium B", "Los Angeles", "CA", "USA"}, EventDate: "2024-12-05"},
		{ID: "3", Name: "Concert C", Venue: Venue{"v3", "Stadium C", "Chicago", "IL", "USA"}, EventDate: "2024-12-10"},
		{ID: "4", Name: "Concert D", Venue: Venue{"v4", "Stadium D", "Houston", "TX", "USA"}, EventDate: "2024-12-15"},
		{ID: "5", Name: "Concert E", Venue: Venue{"v5", "Stadium E", "Phoenix", "AZ", "USA"}, EventDate: "2024-12-20"},
		{ID: "6", Name: "Concert F", Venue: Venue{"v6", "Stadium F", "Philadelphia", "PA", "USA"}, EventDate: "2024-12-25"},
		{ID: "7", Name: "Concert G", Venue: Venue{"v7", "Stadium G", "San Antonio", "TX", "USA"}, EventDate: "2024-12-30"},
		{ID: "8", Name: "Concert H", Venue: Venue{"v8", "Stadium H", "San Diego", "CA", "USA"}, EventDate: "2025-01-05"},
		{ID: "9", Name: "Concert I", Venue: Venue{"v9", "Stadium I", "Dallas", "TX", "USA"}, EventDate: "2025-01-10"},
		{ID: "10", Name: "Concert J", Venue: Venue{"v10", "Stadium J", "San Jose", "CA", "USA"}, EventDate: "2025-01-15"},
	}

	ctx := context.Background()

	for _, event := range events {
		_, err := client.Index().
			Index("events").
			Id(event.ID).
			BodyJson(event).
			Do(ctx)
		if err != nil {
			log.Fatalf("Error indexing event: %s", err)
		}
	}

	fmt.Println("Successfully indexed 10 events.")
}
