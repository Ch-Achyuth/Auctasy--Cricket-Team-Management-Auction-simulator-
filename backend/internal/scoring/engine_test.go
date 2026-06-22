package scoring_test

import (
	"testing"

	"github.com/ch-achyuth/auctasy/internal/scoring"
	"github.com/stretchr/testify/assert"
)

// ── Batting ───────────────────────────────────────────────────────────────────

func TestBatting_RunsAndBoundaries(t *testing.T) {
	// 40 runs + 4 fours + 2 sixes + 30s milestone + SR bonus
	// SR = 40/30*100 = 133.3 → +2 (130 ≤ sr < 150)
	p := scoring.MatchPerformance{
		Runs: 40, Fours: 4, Sixes: 2,
		BallsFaced: 30, WasDismissed: true,
	}
	pts := scoring.Calculate(p)
	// 40 + 4 + 4 + 4 + 2 = 54
	assert.Equal(t, 54.0, pts.Batting)
	assert.Equal(t, 0.0, pts.Bowling)
	assert.Equal(t, 54.0, pts.Total)
}

func TestBatting_FiftyMilestone(t *testing.T) {
	// 55 runs, SR = 55/40*100 = 137.5 → +2 (130 ≤ sr < 150)
	p := scoring.MatchPerformance{
		Runs: 55, BallsFaced: 40, WasDismissed: true,
	}
	pts := scoring.Calculate(p)
	// 55 + 8 (fifty) + 2 (SR) = 65
	assert.Equal(t, 65.0, pts.Batting)
}

func TestBatting_CenturyMilestone(t *testing.T) {
	// 100 runs, SR = 100/70*100 = 142.9 → +2 (130 ≤ sr < 150)
	p := scoring.MatchPerformance{
		Runs: 100, BallsFaced: 70, WasDismissed: true,
	}
	pts := scoring.Calculate(p)
	// 100 + 16 (century) + 2 (SR) = 118
	assert.Equal(t, 118.0, pts.Batting)
}

func TestBatting_Duck(t *testing.T) {
	// Dismissed for 0 — BallsFaced < 10 so no SR modifier applies
	p := scoring.MatchPerformance{
		Runs: 0, BallsFaced: 1, WasDismissed: true,
	}
	pts := scoring.Calculate(p)
	assert.Equal(t, -2.0, pts.Batting)
}

func TestBatting_DNB(t *testing.T) {
	// Did not bat — WasDismissed=false, all stats 0 → exactly 0 points
	p := scoring.MatchPerformance{WasDismissed: false}
	pts := scoring.Calculate(p)
	assert.Equal(t, 0.0, pts.Batting)
}

// Fractional SR penalty — verifies the 50/60/70 tier boundaries.
func TestBatting_FractionalStrikeRatePenalty_Below50(t *testing.T) {
	// SR = 4/10*100 = 40 → < 50 → -6
	p := scoring.MatchPerformance{
		Runs: 4, BallsFaced: 10, WasDismissed: true,
	}
	pts := scoring.Calculate(p)
	// 4 - 6 = -2
	assert.Equal(t, -2.0, pts.Batting)
}

func TestBatting_FractionalStrikeRatePenalty_Below60(t *testing.T) {
	// SR = 5/10*100 = 50 → 50 ≤ sr < 60 → -4
	p := scoring.MatchPerformance{
		Runs: 5, BallsFaced: 10, WasDismissed: true,
	}
	pts := scoring.Calculate(p)
	// 5 - 4 = 1
	assert.Equal(t, 1.0, pts.Batting)
}

func TestBatting_FractionalStrikeRatePenalty_Below70(t *testing.T) {
	// SR = 6/10*100 = 60 → 60 ≤ sr < 70 → -2
	p := scoring.MatchPerformance{
		Runs: 6, BallsFaced: 10, WasDismissed: true,
	}
	pts := scoring.Calculate(p)
	// 6 - 2 = 4
	assert.Equal(t, 4.0, pts.Batting)
}

func TestBatting_StrikeRateBonus_Above170(t *testing.T) {
	// SR = 18/10*100 = 180 → ≥ 170 → +6
	p := scoring.MatchPerformance{
		Runs: 18, BallsFaced: 10, WasDismissed: false,
	}
	pts := scoring.Calculate(p)
	// 18 + 6 = 24
	assert.Equal(t, 24.0, pts.Batting)
}

// ── Bowling ───────────────────────────────────────────────────────────────────

func TestBowling_ThreeWickets_NeutralEconomy(t *testing.T) {
	// 3 wickets + 3wkt bonus; 3 overs (18 balls), 24 runs → econ 8.0 → no bonus/penalty
	p := scoring.MatchPerformance{
		Wickets: 3, BallsBowled: 18, RunsConceded: 24,
	}
	pts := scoring.Calculate(p)
	// 75 + 4 = 79
	assert.Equal(t, 79.0, pts.Bowling)
}

func TestBowling_FiveWickets_ExcellentEconomy(t *testing.T) {
	// 5 wickets + 5wkt bonus; 4 overs (24 balls), 16 runs → econ 4.0 → +6
	p := scoring.MatchPerformance{
		Wickets: 5, BallsBowled: 24, RunsConceded: 16,
	}
	pts := scoring.Calculate(p)
	// 125 + 16 + 6 = 147
	assert.Equal(t, 147.0, pts.Bowling)
}

// Fractional economy penalty — verifies the econ ≥ 12 tier.
func TestBowling_EconomyPenalty_AtExactly12(t *testing.T) {
	// 0 wickets; 2 overs (12 balls), 24 runs → econ = 24/12*6 = 12.0 → -6
	p := scoring.MatchPerformance{
		Wickets: 0, BallsBowled: 12, RunsConceded: 24,
	}
	pts := scoring.Calculate(p)
	assert.Equal(t, -6.0, pts.Bowling)
}

func TestBowling_EconomyPenalty_Between10And11(t *testing.T) {
	// econ = 21/12*6 = 10.5 → ≥ 10, < 11 → -2
	p := scoring.MatchPerformance{
		Wickets: 0, BallsBowled: 12, RunsConceded: 21,
	}
	pts := scoring.Calculate(p)
	assert.Equal(t, -2.0, pts.Bowling)
}

func TestBowling_EconomyBonus_Below6(t *testing.T) {
	// econ = 11/12*6 = 5.5 → < 6 → +4
	p := scoring.MatchPerformance{
		Wickets: 1, BallsBowled: 12, RunsConceded: 11,
	}
	pts := scoring.Calculate(p)
	// 25 + 4 = 29
	assert.Equal(t, 29.0, pts.Bowling)
}

func TestBowling_NoBowling(t *testing.T) {
	p := scoring.MatchPerformance{BallsBowled: 0}
	pts := scoring.Calculate(p)
	assert.Equal(t, 0.0, pts.Bowling)
}

// Less than 2 overs bowled — economy modifier should not apply.
func TestBowling_LessThanTwoOvers_NoEconomyModifier(t *testing.T) {
	// 1 over (6 balls) — below the 12-ball threshold
	p := scoring.MatchPerformance{
		Wickets: 0, BallsBowled: 6, RunsConceded: 12,
	}
	pts := scoring.Calculate(p)
	assert.Equal(t, 0.0, pts.Bowling)
}

// ── Fielding ──────────────────────────────────────────────────────────────────

func TestFielding_ThreeCatches(t *testing.T) {
	p := scoring.MatchPerformance{Catches: 3}
	pts := scoring.Calculate(p)
	assert.Equal(t, 24.0, pts.Fielding)
	assert.Equal(t, 24.0, pts.Total)
}

// ── All-round combined ────────────────────────────────────────────────────────

func TestAllRound_CombinedPoints(t *testing.T) {
	// Batting: 35 + 3 (fours) + 2 (1 six) + 4 (30s milestone)
	//          SR = 35/22*100 = 159.1 → +4 (150 ≤ sr < 170) → 48
	// Bowling: 2*25 = 50; econ = 17/18*6 = 5.67 → +4 (< 6) → 54
	// Fielding: 1 catch → 8
	// Total: 110
	p := scoring.MatchPerformance{
		Runs: 35, Fours: 3, Sixes: 1, BallsFaced: 22, WasDismissed: true,
		Wickets: 2, BallsBowled: 18, RunsConceded: 17,
		Catches: 1,
	}
	pts := scoring.Calculate(p)
	assert.Equal(t, 48.0, pts.Batting)
	assert.Equal(t, 54.0, pts.Bowling)
	assert.Equal(t, 8.0, pts.Fielding)
	assert.Equal(t, 110.0, pts.Total)
}
