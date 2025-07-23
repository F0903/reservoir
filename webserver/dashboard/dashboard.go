package dashboard

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
)

var (
	ErrFrontendNotFound = errors.New("frontend files not found")
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
		return fmt.Errorf("%w: %v", ErrFrontendNotFound, err)
	}
	mux.Handle("/", http.FileServer(http.FS(frontendDist)))

	return nil
}
