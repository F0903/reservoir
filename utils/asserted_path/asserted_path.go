package asserted_path

import (
	"errors"
	"log"
	"os"
)

type AssertedPath struct {
	// Path is the asserted path.
	path string
}

func assertDir(dir string) {
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		return // Directory already exists, no need to create it
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Panicf("failed to create directories '%s': %v", dir, err)
	}
}

func Assert(path string) *AssertedPath {
	return &AssertedPath{
		path: path,
	}
}

func (ap *AssertedPath) GetPath() string {
	assertDir(ap.path)
	return ap.path
}

func (ap *AssertedPath) String() string {
	return ap.GetPath()
}
