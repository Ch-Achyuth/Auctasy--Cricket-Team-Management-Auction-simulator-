package scoring

// MatchPerformance holds a player's raw stats for a single T20 innings.
// All counts are non-negative; zero values mean the player did not bat / bowl.
type MatchPerformance struct {
	// Batting
	Runs         int
	Fours        int
	Sixes        int
	BallsFaced   int
	WasDismissed bool // false for DNB or not-out

	// Bowling
	Wickets      int
	RunsConceded int
	BallsBowled  int // 0 means the player did not bowl

	// Fielding
	Catches int
}

// Points is the fantasy-point breakdown for one player in one match.
type Points struct {
	Total    float64
	Batting  float64
	Bowling  float64
	Fielding float64
}

// Calculate returns the IPL-style fantasy points for the given match performance.
func Calculate(p MatchPerformance) Points {
	var pts Points
	pts.Batting = battingPoints(p)
	pts.Bowling = bowlingPoints(p)
	pts.Fielding = fieldingPoints(p)
	pts.Total = pts.Batting + pts.Bowling + pts.Fielding
	return pts
}

func battingPoints(p MatchPerformance) float64 {
	pts := float64(p.Runs)           // 1 pt per run
	pts += float64(p.Fours)          // 1 pt per boundary four
	pts += float64(p.Sixes) * 2      // 2 pts per six

	// Milestone bonus (only the highest milestone is awarded)
	switch {
	case p.Runs >= 100:
		pts += 16
	case p.Runs >= 50:
		pts += 8
	case p.Runs >= 30:
		pts += 4
	}

	// Duck penalty — dismissed for zero
	if p.WasDismissed && p.Runs == 0 {
		pts -= 2
	}

	// Strike-rate bonus / penalty — minimum 10 balls faced to qualify
	if p.BallsFaced >= 10 {
		sr := float64(p.Runs) / float64(p.BallsFaced) * 100
		switch {
		case sr >= 170:
			pts += 6
		case sr >= 150:
			pts += 4
		case sr >= 130:
			pts += 2
		case sr < 50:
			pts -= 6
		case sr < 60:
			pts -= 4
		case sr < 70:
			pts -= 2
		}
	}

	return pts
}

func bowlingPoints(p MatchPerformance) float64 {
	if p.BallsBowled == 0 {
		return 0
	}

	pts := float64(p.Wickets) * 25

	// Wicket haul bonus
	switch {
	case p.Wickets >= 5:
		pts += 16
	case p.Wickets >= 4:
		pts += 8
	case p.Wickets >= 3:
		pts += 4
	}

	// Economy-rate bonus / penalty — minimum 2 overs (12 balls) to qualify
	if p.BallsBowled >= 12 {
		econ := float64(p.RunsConceded) / float64(p.BallsBowled) * 6
		switch {
		case econ < 5:
			pts += 6
		case econ < 6:
			pts += 4
		case econ < 7:
			pts += 2
		case econ >= 12:
			pts -= 6
		case econ >= 11:
			pts -= 4
		case econ >= 10:
			pts -= 2
		}
	}

	return pts
}

func fieldingPoints(p MatchPerformance) float64 {
	return float64(p.Catches) * 8
}
