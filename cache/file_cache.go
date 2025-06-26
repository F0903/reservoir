package cache

import (
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var hasher = crypto.BLAKE2b_256.New()

type FileCache[ObjectData any] struct {
	rootDir string
	locks   map[string]*sync.RWMutex
	mu      sync.Mutex // protects locks map
}

// NewFileCache creates a new FileCache instance with the specified root directory.
func NewFileCache[ObjectData any](rootDir string) *FileCache[ObjectData] {
	return &FileCache[ObjectData]{
		rootDir: rootDir,
		locks:   make(map[string]*sync.RWMutex),
	}
}

func getMetaPath(dataPath string) string {
	return dataPath[:len(dataPath)-len(filepath.Ext(dataPath))] + ".meta"
}

// getLock returns the per-file lock for a given file name (creates if missing)
func (c *FileCache[ObjectData]) getLock(fileName string) *sync.RWMutex {
	c.mu.Lock()
	defer c.mu.Unlock()

	lock, ok := c.locks[fileName]
	if !ok {
		lock = &sync.RWMutex{}
		c.locks[fileName] = lock
	}

	return lock
}

func (c *FileCache[ObjectData]) Get(input string) (*Entry[ObjectData], error) {
	inputHash := string(hasher.Sum([]byte(input)))

	lock := c.getLock(inputHash)
	lock.RLock()
	defer lock.RUnlock()

	fileName := filepath.Join(c.rootDir, inputHash)
	if _, err := os.Stat(fileName); errors.Is(err, os.ErrNotExist) {
		return nil, ErrorCacheMiss
	}

	dataStream, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read cached data file '%s': %v", fileName, err)
	}

	metaPath := getMetaPath(fileName)
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cached metadata file '%s': %v", metaPath, err)
	}

	var meta EntryMetadata[ObjectData]
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata from '%s': %v", metaPath, err)
	}

	return &Entry[ObjectData]{
		Data:     dataStream,
		Metadata: meta,
	}, nil
}

func (c *FileCache[ObjectData]) Cache(input string, data io.Reader, expires time.Time, objectData ObjectData) error {
	inputHash := string(hasher.Sum([]byte(input)))

	lock := c.getLock(inputHash)
	lock.Lock()
	defer lock.Unlock()

	fileName := filepath.Join(c.rootDir, inputHash)
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create cache file '%s': %v", fileName, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, data); err != nil {
		return fmt.Errorf("failed to write cache file '%s': %v", fileName, err)
	}

	metaPath := getMetaPath(fileName)
	metaFile, err := os.Create(metaPath)
	if err != nil {
		return fmt.Errorf("failed to create cache metadata file '%s': %v", metaPath, err)
	}
	defer metaFile.Close()

	metaJson, err := json.Marshal(EntryMetadata[ObjectData]{
		TimeWritten: time.Now(),
		Expires:     expires,
		Object:      objectData,
	})
	if err != nil {
		return fmt.Errorf("failed to encode json metadata for '%s': %v", metaPath, err)
	}

	if _, err := metaFile.Write(metaJson); err != nil {
		return fmt.Errorf("failed to write cache metadata file '%s': %v", metaPath, err)
	}

	return nil
}

func (c *FileCache[ObjectData]) Delete(input string) error {
	inputHash := string(hasher.Sum([]byte(input)))

	lock := c.getLock(inputHash)
	lock.Lock()
	defer lock.Unlock()

	fileName := filepath.Join(c.rootDir, inputHash)
	if err := os.Remove(fileName); err != nil {
		os.Remove(getMetaPath(fileName)) // Ensure no orphaned metadata file exists
		return fmt.Errorf("failed to delete cache file '%s': %v", fileName, err)
	}

	metaPath := getMetaPath(fileName)
	if err := os.Remove(metaPath); err != nil {
		return fmt.Errorf("failed to delete cache metadata file '%s': %v", metaPath, err)
	}

	return nil
}
