package asserted_path

import (
	"errors"
	"log"
	"os"
	"path/filepath"
)

type AssertedPath struct {
	// Path is the asserted path.
	path string
}

func assertPath(path string) {
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		return // Path already exists, no need to create it
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Panicf("failed to create directories '%s': %v", path, err)
	}
}

func Assert(path string) AssertedPath {
	return AssertedPath{
		path: path,
	}
}

func (ap *AssertedPath) GetPath() string {
	assertPath(ap.path)
	return ap.path
}

func (ap *AssertedPath) String() string {
	return ap.GetPath()
}
