package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/olivere/elastic/v7"
)

type SearchRequest struct {
	Query    string `json:"query"`
	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
}

func SearchEvents(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client, err := elastic.NewClient(
		elastic.SetURL("http://localhost:9200"),
		elastic.SetSniff(false),
	)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	query := elastic.NewBoolQuery()

	// Full-text search on name and venue
	if req.Query != "" {
		query.Must(elastic.NewMultiMatchQuery(req.Query, "name", "venue.name"))
	}

	// Date range filter
	if req.DateFrom != "" && req.DateTo != "" {
		query.Filter(elastic.NewRangeQuery("event_date").Gte(req.DateFrom).Lte(req.DateTo))
	}

	searchResult, err := client.Search().
		Index("events").
		Query(query).
		Do(context.Background())

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searchResult.Hits.Hits)
}
