package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ch-achyuth/auctasy/internal/database"
	"github.com/ch-achyuth/auctasy/internal/middleware"
)

type placeBidBody struct {
	AuctionQueueID string `json:"auction_queue_id"`
	LeagueID       string `json:"league_id"`
	BidAmount      int64  `json:"bid_amount"`
}

// PlaceBid handles POST /api/v1/auction/bid.
//
// The caller must supply a valid Supabase JWT (enforced by RequireAuth middleware).
// The handler extracts the authenticated user ID from context, then calls the
// place_bid Postgres RPC which atomically validates and records the bid in a
// single database transaction.
func PlaceBid(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID, ok := r.Context().Value(middleware.CtxUserID).(string)
		if !ok || userID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var body placeBidBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if body.AuctionQueueID == "" || body.LeagueID == "" || body.BidAmount <= 0 {
			http.Error(w, "auction_queue_id, league_id and bid_amount are required", http.StatusBadRequest)
			return
		}

		params, err := json.Marshal(map[string]interface{}{
			"p_user_id":          userID,
			"p_auction_queue_id": body.AuctionQueueID,
			"p_league_id":        body.LeagueID,
			"p_bid_amount":       body.BidAmount,
		})
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		result, err := db.CallRPC(r.Context(), "place_bid", params)
		if err != nil {
			log.Printf("place_bid rpc error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// The RPC returns a jsonb object; forward it verbatim to the client.
		w.Header().Set("Content-Type", "application/json")
		w.Write(result)
	}
}
