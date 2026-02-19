package dashboard

import (
	"embed"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"reservoir/config"
	"strings"
	"time"
)

var (
	ErrFrontendNotFound = errors.New("frontend files not found")
)

//go:embed frontend/build/*
var frontend embed.FS
var buildFS = func() fs.FS {
	fsys, err := fs.Sub(frontend, "frontend/build")
	if err != nil {
		panic(ErrFrontendNotFound)
	}
	return fsys
}()

type Dashboard struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Dashboard {
	return &Dashboard{cfg: cfg}
}

func (d *Dashboard) ServeDashboard(w http.ResponseWriter, r *http.Request) {
	fName := strings.TrimPrefix(r.URL.Path, "/")
	if fName == "" {
		fName = "index.html"
	}

	// Try to open the requested file, if it doesn't exist serve index.html (for SPA routing)
	if f, err := buildFS.Open(fName); err == nil {
		defer f.Close()
		http.ServeContent(w, r, fName, time.Time{}, f.(io.ReadSeeker))
		return
	}

	index, err := buildFS.Open("index.html")
	if err != nil {
		http.Error(w, "frontend missing index.html", http.StatusInternalServerError)
		return
	}
	defer index.Close()

	http.ServeContent(w, r, fName, time.Time{}, index.(io.ReadSeeker))
}

func (d *Dashboard) RegisterHandlers(mux *http.ServeMux) error {
	mux.HandleFunc("/", d.ServeDashboard)

	return nil
}
