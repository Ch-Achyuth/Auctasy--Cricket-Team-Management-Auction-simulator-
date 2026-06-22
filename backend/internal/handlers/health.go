package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// Health responds with a JSON status object. Used by load-balancer health checks
// and the /health liveness probe.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}
