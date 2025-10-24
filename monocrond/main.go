package main

import (
	"net"
	"net/http"
	"os"

	"github.com/Omotolani98/monocrond/config"
	"github.com/Omotolani98/monocrond/handler"
	"github.com/charmbracelet/log"
)

func main() {
	log.Info("Welcome to Monocrond")
	sock, err := net.Listen("unix", config.SocketFile)
	if err != nil {
		log.Errorf("Error on socket connection <|::|> %v", err)
		panic(err)
	}

	defer func() { _ = sock.Close(); _ = os.Remove(config.SocketFile) }()
	config.StartCron()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handler.Health)
	mux.HandleFunc("/schedule", handler.Schedule)
	mux.HandleFunc("/shutdown", handler.Shutdown)
	mux.HandleFunc("/list", handler.ListJobs)
	mux.HandleFunc("/jobs/", handler.GetJob)
	mux.HandleFunc("/delete/", handler.DeleteJob)

	srv := &http.Server{Handler: mux}
	if err := srv.Serve(sock); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
