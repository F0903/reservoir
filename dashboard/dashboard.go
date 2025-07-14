package dashboard

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed frontend/build/*
var frontend embed.FS

type Dashboard struct {
	mux *http.ServeMux
}

func Init() *Dashboard {
	mux := http.NewServeMux()

	//TODO: Add API endpoints for the dashboard
	// mux.HandleFunc("/api/stats", p.handleStats)

	// Serve Svelte frontend files
	frontendDist, err := fs.Sub(frontend, "frontend/build")
	if err != nil {
		log.Fatalf("Failed to create subdirectory for frontend/build: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(frontendDist)))

	return &Dashboard{
		mux: mux,
	}
}

func (d *Dashboard) ListenBlocking(address string) error {
	log.Println("Starting dashboard server on", address)
	return http.ListenAndServe(address, d.mux)
}

func (d *Dashboard) Listen(address string) {
	go func() {
		if err := d.ListenBlocking(address); err != nil {
			log.Println("Error during non-blocking listen:", err)
		}
	}()
}
