package dashboard

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

//go:embed frontend/build/*
var frontend embed.FS

type Dashboard struct{}

func New() *Dashboard {
	return &Dashboard{}
}

func (d *Dashboard) RegisterHandlers(mux *http.ServeMux) error {
	frontendDist, err := fs.Sub(frontend, "frontend/build")
	if err != nil {
		return fmt.Errorf("Failed to create subdirectory for frontend/build: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(frontendDist)))

	return nil
}
