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
    w.Header().Set("Content-Type", "application/json; charset=utf-8")

    entries := config.Cron.Entries()
    out := make([]models.EntryView, 0, len(entries))

    config.Mu.RLock()
    for _, e := range entries {
        name := ""
        spec := ""
        if meta, ok := config.Jobs[e.ID]; ok {
            name = meta.Name
            spec = meta.Spec
        }
        out = append(out, models.EntryView{
            ID:   int(e.ID),
            Name: name,
            Schedule: spec,
			Description: "",
            Next: e.Next,
            Prev: e.Prev,
        })
    }
    config.Mu.RUnlock()

    if err := json.NewEncoder(w).Encode(out); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
