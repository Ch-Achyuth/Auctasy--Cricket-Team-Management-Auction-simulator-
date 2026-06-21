package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ch-achyuth/auctasy/internal/config"
	"github.com/ch-achyuth/auctasy/internal/database"
	"github.com/joho/godotenv"
)

type Player struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

var playerCache = make(map[string]string)

// PlayerHistoricalStats matches our new table.
type PlayerHistoricalStats struct {
	PlayerID string `json:"player_id"`
	Format   string `json:"format"`

	Matches int `json:"matches,omitempty"`

	BattingInns int     `json:"batting_inns,omitempty"`
	BattingNO   int     `json:"batting_no,omitempty"`
	BattingRuns int     `json:"batting_runs,omitempty"`
	BattingHS   string  `json:"batting_hs,omitempty"`
	BattingAve  float64 `json:"batting_ave,omitempty"`
	BattingBF   int     `json:"batting_bf,omitempty"`
	BattingSR   float64 `json:"batting_sr,omitempty"`
	Batting100s int     `json:"batting_100s,omitempty"`
	Batting50s  int     `json:"batting_50s,omitempty"`
	Batting4s   int     `json:"batting_4s,omitempty"`
	Batting6s   int     `json:"batting_6s,omitempty"`

	BowlingInns  int     `json:"bowling_inns,omitempty"`
	BowlingBalls int     `json:"bowling_balls,omitempty"`
	BowlingRuns  int     `json:"bowling_runs,omitempty"`
	BowlingWkts  int     `json:"bowling_wkts,omitempty"`
	BowlingBBI   string  `json:"bowling_bbi,omitempty"`
	BowlingAve   float64 `json:"bowling_ave,omitempty"`
	BowlingEcon  float64 `json:"bowling_econ,omitempty"`
	BowlingSR    float64 `json:"bowling_sr,omitempty"`
	Bowling4w    int     `json:"bowling_4w,omitempty"`
	Bowling5w    int     `json:"bowling_5w,omitempty"`
}

func main() {
	_ = godotenv.Load("../.env")
	
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.Connect(cfg.SupabaseURL, cfg.SupabaseKey)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	battingDir := "data/batting_stats"
	fmt.Println("🏏 Processing Batting Stats...")
	filepath.Walk(battingDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".csv") {
			format := extractFormat(info.Name())
			processBattingCSV(db, path, format)
		}
		return nil
	})

	bowlingDir := "data/bowling_stats"
	fmt.Println("⚾ Processing Bowling Stats...")
	filepath.Walk(bowlingDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".csv") {
			format := extractFormat(info.Name())
			processBowlingCSV(db, path, format)
		}
		return nil
	})

	fmt.Println("✅ Data Seeding Complete!")
}

// Extract year from "batting_stats_ipl_2016.csv" to make format "IPL 2016"
func extractFormat(filename string) string {
	re := regexp.MustCompile(`ipl_(\d{4})`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) > 1 {
		return "IPL " + matches[1]
	}
	return "Unknown Format"
}

func getPlayerID(db *database.DB, name string) string {
	// Clean the name (remove leading/trailing spaces)
	name = strings.TrimSpace(name)

	if id, exists := playerCache[name]; exists {
		return id
	}

	dataStr, _, err := db.Client.From("players").Select("id", "exact", false).Eq("name", name).Execute()
	if err != nil {
		log.Printf("Query error for %s: %v", name, err)
		return ""
	}

	var results []Player
	if err := json.Unmarshal([]byte(dataStr), &results); err == nil && len(results) > 0 {
		playerCache[name] = results[0].ID
		return results[0].ID
	}

	// Insert player
	newPlayer := Player{Name: name}
	dataStr, _, err = db.Client.From("players").Insert(newPlayer, false, "", "representation", "exact").Execute()
	if err != nil {
		log.Printf("Insert error for %s: %v", name, err)
		return ""
	}

	var inserted []Player
	if err := json.Unmarshal([]byte(dataStr), &inserted); err == nil && len(inserted) > 0 {
		playerCache[name] = inserted[0].ID
		return inserted[0].ID
	}
	return ""
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "-" || s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "-" || s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func processBattingCSV(db *database.DB, filepath string, format string) {
	file, err := os.Open(filepath)
	if err != nil {
		log.Printf("Error opening %s: %v", filepath, err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Skip header
	_, _ = reader.Read()

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 14 {
			continue
		}

		// POS,Player,Mat,Inns,NO,Runs,HS,Avg,BF,SR,100,50,4s,6s
		playerName := record[1]
		playerID := getPlayerID(db, playerName)
		if playerID == "" {
			continue
		}

		stat := PlayerHistoricalStats{
			PlayerID:    playerID,
			Format:      format,
			Matches:     parseInt(record[2]),
			BattingInns: parseInt(record[3]),
			BattingNO:   parseInt(record[4]),
			BattingRuns: parseInt(record[5]),
			BattingHS:   record[6],
			BattingAve:  parseFloat(record[7]),
			BattingBF:   parseInt(record[8]),
			BattingSR:   parseFloat(record[9]),
			Batting100s: parseInt(record[10]),
			Batting50s:  parseInt(record[11]),
			Batting4s:   parseInt(record[12]),
			Batting6s:   parseInt(record[13]),
		}

		upsertStats(db, stat)
	}
	fmt.Printf("Processed Batting: %s\n", filepath)
}

// Convert "4.3" overs to 27 balls
func parseOversToBalls(ovStr string) int {
	ovStr = strings.TrimSpace(ovStr)
	if ovStr == "-" || ovStr == "" {
		return 0
	}
	parts := strings.Split(ovStr, ".")
	overs := parseInt(parts[0])
	balls := 0
	if len(parts) > 1 {
		balls = parseInt(parts[1])
	}
	return (overs * 6) + balls
}

func processBowlingCSV(db *database.DB, filepath string, format string) {
	file, err := os.Open(filepath)
	if err != nil {
		log.Printf("Error opening %s: %v", filepath, err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Skip header
	_, _ = reader.Read()

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 13 {
			continue
		}

		// POS,Player,Mat,Inns,Ov,Runs,Wkts,BBI,Avg,Econ,SR,4w,5w
		playerName := record[1]
		playerID := getPlayerID(db, playerName)
		if playerID == "" {
			continue
		}

		stat := PlayerHistoricalStats{
			PlayerID:     playerID,
			Format:       format,
			Matches:      parseInt(record[2]), // Updates if it's the same/higher
			BowlingInns:  parseInt(record[3]),
			BowlingBalls: parseOversToBalls(record[4]),
			BowlingRuns:  parseInt(record[5]),
			BowlingWkts:  parseInt(record[6]),
			BowlingBBI:   record[7],
			BowlingAve:   parseFloat(record[8]),
			BowlingEcon:  parseFloat(record[9]),
			BowlingSR:    parseFloat(record[10]),
			Bowling4w:    parseInt(record[11]),
			Bowling5w:    parseInt(record[12]),
		}

		upsertStats(db, stat)
	}
	fmt.Printf("Processed Bowling: %s\n", filepath)
}

func upsertStats(db *database.DB, stat PlayerHistoricalStats) {
	// PostgREST upsert requires the row data and onConflict parameter
	_, _, err := db.Client.From("player_historical_stats").Insert(stat, true, "player_id,format", "representation", "exact").Execute()
	if err != nil {
		log.Printf("Error upserting stats for player %s format %s: %v", stat.PlayerID, stat.Format, err)
	}
}
