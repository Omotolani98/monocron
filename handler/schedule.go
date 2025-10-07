package handler

import (
	"encoding/json"
	"net/http"

	"github.com/charmbracelet/log"
)

func Schedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	var data map[string]any
	json.NewDecoder(r.Body).Decode(&data)
	log.Info("Received JSON:", data)

	resp := map[string]string{"status": "ok", "msg": "job scheduled"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
