package handler

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/Omotolani98/monocrond/config"
	cmdUtil "github.com/Omotolani98/monocrond/internal"
	"github.com/Omotolani98/monocrond/models"
	"github.com/charmbracelet/log"
)

func Schedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	var data models.ScheduleRequest
	json.NewDecoder(r.Body).Decode(&data)
	log.Info("Received JSON:", data)

	resp, _ := cmdUtil.ScheduleCron(data)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func Shutdown(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    go func() {
        stopCtx := config.Cron.Stop()
        <-stopCtx.Done()
        _ = os.Remove(config.TempFile)
        os.Exit(0)
    }()
}

func ListJobs(w http.ResponseWriter, r *http.Request) {
	log.Info(config.Cron.Entries())
	json.NewEncoder(w).Encode(config.Cron.Entries())
}
