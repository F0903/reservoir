package assertedpath

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type AssertedPath struct {
	Path string
}

func createDirs(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directories '%s': %v", path, err)
	}
	return nil
}

func assertPath(path string) {
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		return // Path already exists, no need to create it
	}

	if err := createDirs(path); err != nil {
		log.Panic(err)
	}
}

func Assert(path string) AssertedPath {
	assertPath(path)
	return AssertedPath{
		Path: path,
	}
}

func (ap AssertedPath) EnsureCleared() AssertedPath {
	// Just remove the path and recreate it, simpler and faster than iterating through the directory
	os.RemoveAll(ap.Path)
	if err := createDirs(ap.Path); err != nil {
		log.Panic(err)
	}
	return ap
}

func (ap *AssertedPath) String() string {
	return ap.Path
}
