package auction_test

import (
	"testing"
	"time"

	"github.com/ch-achyuth/auctasy/internal/auction"
	"github.com/stretchr/testify/assert"
)

// base returns a valid bid request that all tests mutate from a clean baseline.
func base() auction.BidRequest {
	return auction.BidRequest{
		Amount:         600,
		UserBudget:     1000,
		CurrentHighBid: 500,
		BasePrice:      500,
		ItemStatus:     "active",
	}
}

func TestValidateBid_Normal(t *testing.T) {
	assert.NoError(t, auction.ValidateBid(base()))
}

// ── Budget edge cases ─────────────────────────────────────────────────────────

// A bid exactly equal to the user's remaining budget must succeed.
// This is the most important budget boundary: the rule is amount > budget fails,
// amount == budget passes.
func TestValidateBid_ExactBudget(t *testing.T) {
	req := base()
	req.Amount = req.UserBudget // 1000 == 1000 → allowed
	assert.NoError(t, auction.ValidateBid(req))
}

// One unit above budget must fail.
func TestValidateBid_BudgetExceededByOne(t *testing.T) {
	req := base()
	req.Amount = req.UserBudget + 1
	assert.ErrorIs(t, auction.ValidateBid(req), auction.ErrInsufficientBudget)
}

// ── First-bid edge cases ──────────────────────────────────────────────────────

// The very first bid on an item at exactly the base price must succeed.
func TestValidateBid_FirstBidEqualsBasePrice(t *testing.T) {
	req := base()
	req.CurrentHighBid = 0
	req.Amount = req.BasePrice
	assert.NoError(t, auction.ValidateBid(req))
}

// First bid one unit below base price must fail.
func TestValidateBid_FirstBidBelowBasePrice(t *testing.T) {
	req := base()
	req.CurrentHighBid = 0
	req.Amount = req.BasePrice - 1
	assert.ErrorIs(t, auction.ValidateBid(req), auction.ErrBidBelowBase)
}

// ── Out-of-order / stale bid simulation ──────────────────────────────────────

// Simulates a bid that was valid when it left the client but arrived late:
// another user already pushed the high bid to 900, so the 700 bid is stale.
func TestValidateBid_StaleOutOfOrderBid(t *testing.T) {
	req := base()
	req.CurrentHighBid = 900
	req.Amount = 700 // arrived out of order — lower than the current leader
	assert.ErrorIs(t, auction.ValidateBid(req), auction.ErrBidNotHighEnough)
}

// A bid exactly equal to the current high (not higher) must also fail.
func TestValidateBid_BidEqualToCurrentHigh(t *testing.T) {
	req := base()
	req.Amount = req.CurrentHighBid
	assert.ErrorIs(t, auction.ValidateBid(req), auction.ErrBidNotHighEnough)
}

// ── Auction clock edge cases ──────────────────────────────────────────────────

// A bid arriving 1 nanosecond after the deadline must be rejected.
// This is the "zero seconds left" scenario: the clock expired before the bid arrived.
func TestValidateBid_ArrivesOneNsAfterDeadline(t *testing.T) {
	deadline := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	req := base()
	req.EndsAt = deadline
	req.Now = deadline.Add(time.Nanosecond) // strictly after → expired
	assert.ErrorIs(t, auction.ValidateBid(req), auction.ErrAuctionExpired)
}

// A bid that arrives exactly at the deadline (Now == EndsAt) is NOT expired —
// After() is strictly greater than, so Now == EndsAt returns false.
func TestValidateBid_ArrivesAtExactDeadline(t *testing.T) {
	deadline := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	req := base()
	req.EndsAt = deadline
	req.Now = deadline // equal, not after → still valid
	assert.NoError(t, auction.ValidateBid(req))
}

// A bid arriving 1 nanosecond before the deadline must succeed.
func TestValidateBid_ArrivesJustBeforeDeadline(t *testing.T) {
	deadline := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	req := base()
	req.EndsAt = deadline
	req.Now = deadline.Add(-time.Nanosecond)
	assert.NoError(t, auction.ValidateBid(req))
}

// ── Item status edge cases ────────────────────────────────────────────────────

func TestValidateBid_ItemAlreadySold(t *testing.T) {
	req := base()
	req.ItemStatus = "sold"
	assert.ErrorIs(t, auction.ValidateBid(req), auction.ErrItemNotActive)
}

func TestValidateBid_ItemUnsold(t *testing.T) {
	req := base()
	req.ItemStatus = "unsold"
	assert.ErrorIs(t, auction.ValidateBid(req), auction.ErrItemNotActive)
}

// ── No time limit ─────────────────────────────────────────────────────────────

// When EndsAt is zero the auction has no time limit; a valid bid must pass.
func TestValidateBid_NoTimeLimitActive(t *testing.T) {
	req := base()
	req.EndsAt = time.Time{} // zero value
	assert.NoError(t, auction.ValidateBid(req))
}
