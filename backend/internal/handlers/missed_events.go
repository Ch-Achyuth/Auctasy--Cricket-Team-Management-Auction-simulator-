package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/ch-achyuth/auctasy/internal/database"
)

// MissedEvents handles GET /api/v1/auction/missed-events.
//
// Query params:
//
//	league_id  — UUID of the league room (required)
//	after_seq  — last seq the client has seen; returns events with seq > this (default 0)
//
// Looks up the active auction for the league, then calls the get_missed_events
// stored procedure to return all WAL events the client missed.
func MissedEvents(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		leagueID := r.URL.Query().Get("league_id")
		if leagueID == "" {
			http.Error(w, "league_id is required", http.StatusBadRequest)
			return
		}

		afterSeq := int64(0)
		if s := r.URL.Query().Get("after_seq"); s != "" {
			var err error
			afterSeq, err = strconv.ParseInt(s, 10, 64)
			if err != nil {
				http.Error(w, "after_seq must be an integer", http.StatusBadRequest)
				return
			}
		}

		auctionID, err := db.ActiveAuctionID(r.Context(), leagueID)
		if err != nil {
			http.Error(w, "no active auction", http.StatusNotFound)
			return
		}

		params, err := json.Marshal(map[string]interface{}{
			"p_auction_id": auctionID,
			"p_after_seq":  afterSeq,
		})
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		result, err := db.CallRPC(r.Context(), "get_missed_events", params)
		if err != nil {
			log.Printf("get_missed_events rpc error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(result)
	}
}
