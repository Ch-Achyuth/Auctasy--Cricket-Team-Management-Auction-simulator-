package auction

import (
	"errors"
	"time"
)

var (
	ErrBidBelowBase       = errors.New("bid below base price")
	ErrBidNotHighEnough   = errors.New("bid must exceed current highest bid")
	ErrInsufficientBudget = errors.New("insufficient budget")
	ErrItemNotActive      = errors.New("auction item is not active")
	ErrAuctionExpired     = errors.New("auction time has expired")
)

// BidRequest carries all data needed to validate a bid without a database round-trip.
// The corresponding SQL validation in place_bid is the authoritative check;
// this struct lets unit tests and the API layer perform a fast pre-flight check.
type BidRequest struct {
	Amount         int64
	UserBudget     int64
	CurrentHighBid int64     // 0 means no bids have been placed yet
	BasePrice      int64
	ItemStatus     string    // "active" | "sold" | "unsold"
	EndsAt         time.Time // zero value → no timer enforced
	Now            time.Time // injectable clock; zero value → use real time
}

// ValidateBid returns nil if the bid satisfies all game rules, or a typed error
// describing the first violated constraint.
func ValidateBid(req BidRequest) error {
	if req.ItemStatus != "active" {
		return ErrItemNotActive
	}

	now := req.Now
	if now.IsZero() {
		now = time.Now()
	}
	// A bid arriving at exactly EndsAt is expired; strictly after the deadline.
	if !req.EndsAt.IsZero() && now.After(req.EndsAt) {
		return ErrAuctionExpired
	}

	if req.CurrentHighBid == 0 {
		// First bid on this item — must meet the base price
		if req.Amount < req.BasePrice {
			return ErrBidBelowBase
		}
	} else {
		// Subsequent bid — must strictly exceed the current leader
		if req.Amount <= req.CurrentHighBid {
			return ErrBidNotHighEnough
		}
	}

	if req.Amount > req.UserBudget {
		return ErrInsufficientBudget
	}

	return nil
}
