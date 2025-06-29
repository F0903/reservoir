package cache

import (
	"apt_cacher_go/utils/asserted_path"
	"apt_cacher_go/utils/syncmap"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileCache[ObjectData any] struct {
	rootDir *asserted_path.AssertedPath
	locks   *syncmap.SyncMap[string, *sync.RWMutex]
}

// NewFileCache creates a new FileCache instance with the specified root directory.
func NewFileCache[ObjectData any](rootDir string) *FileCache[ObjectData] {
	return &FileCache[ObjectData]{
		rootDir: asserted_path.Assert(rootDir),
		locks:   syncmap.NewSyncMap[string, *sync.RWMutex](),
	}
}

func getMetaPath(dataPath string) string {
	return dataPath[:len(dataPath)-len(filepath.Ext(dataPath))] + ".meta"
}

func (c *FileCache[ObjectData]) Get(key *CacheKey) (*Entry[ObjectData], error) {
	lock := c.locks.GetOrSet(key.Hex(), &sync.RWMutex{})
	lock.RLock()
	defer lock.RUnlock()

	fileName := filepath.Join(c.rootDir.GetPath(), key.Hex())
	if _, err := os.Stat(fileName); errors.Is(err, os.ErrNotExist) {
		return nil, ErrorCacheMiss
	}

	dataFile, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read cached data file '%s': %v", fileName, err)
	}
	// We don't close dataFile here since we are returning it in the Entry.

	metaPath := getMetaPath(fileName)
	metaFile, err := os.Open(metaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cached metadata file '%s': %v", metaPath, err)
	}
	defer metaFile.Close()

	var meta EntryMetadata[ObjectData]
	if err := json.NewDecoder(metaFile).Decode(&meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata from '%s': %v", metaPath, err)
	}

	stale := false
	if !meta.Expires.IsZero() && meta.Expires.Before(time.Now()) {
		stale = true // The entry is stale if the expiration time is in the past
	}

	return &Entry[ObjectData]{
		Data:     dataFile,
		Metadata: meta,
		Stale:    stale,
	}, nil
}

func (c *FileCache[ObjectData]) Cache(key *CacheKey, data io.Reader, expires time.Time, objectData ObjectData) (*Entry[ObjectData], error) {
	lock := c.locks.GetOrSet(key.Hex(), &sync.RWMutex{})
	lock.Lock()
	defer lock.Unlock()

	fileName := filepath.Join(c.rootDir.GetPath(), key.Hex())
	file, err := os.Create(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache file '%s': %v", fileName, err)
	}
	// We don't close file here since we are returning it in the Entry.

	if _, err := io.Copy(file, data); err != nil {
		return nil, fmt.Errorf("failed to write cache file '%s': %v", fileName, err)
	}

	metaPath := getMetaPath(fileName)
	metaFile, err := os.Create(metaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache metadata file '%s': %v", metaPath, err)
	}
	defer metaFile.Close()

	metaJson, err := json.Marshal(EntryMetadata[ObjectData]{
		TimeWritten: time.Now(),
		Expires:     expires,
		Object:      objectData,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode json metadata for '%s': %v", metaPath, err)
	}

	if _, err := metaFile.Write(metaJson); err != nil {
		return nil, fmt.Errorf("failed to write cache metadata file '%s': %v", metaPath, err)
	}

	file.Seek(0, io.SeekStart) // Reset file stream to the beginning
	return &Entry[ObjectData]{
		Data:     file,
		Metadata: EntryMetadata[ObjectData]{TimeWritten: time.Now(), Expires: expires, Object: objectData},
	}, nil
}

func (c *FileCache[ObjectData]) Delete(key *CacheKey) error {
	lock := c.locks.GetOrSet(key.Hex(), &sync.RWMutex{})
	lock.Lock()
	defer lock.Unlock()

	fileName := filepath.Join(c.rootDir.GetPath(), key.Hex())
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

func (c *FileCache[ObjectData]) UpdateMetadata(key *CacheKey, modifier func(*EntryMetadata[ObjectData])) error {
	lock := c.locks.GetOrSet(key.Hex(), &sync.RWMutex{})
	lock.Lock()
	defer lock.Unlock()

	metaPath := getMetaPath(filepath.Join(c.rootDir.GetPath(), key.Hex()))
	metaFile, err := os.OpenFile(metaPath, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to read cache metadata file '%s': %v", metaPath, err)
	}
	defer metaFile.Close()

	var meta EntryMetadata[ObjectData]
	if err := json.NewDecoder(metaFile).Decode(&meta); err != nil {
		return fmt.Errorf("failed to unmarshal metadata from '%s': %v", metaPath, err)
	}

	modifier(&meta)

	metaJson, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to encode json metadata for '%s': %v", metaPath, err)
	}

	// Clear the file before writing new data
	if err := metaFile.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate cache metadata file '%s': %v", metaPath, err)
	}
	if _, err := metaFile.WriteAt(metaJson, 0); err != nil {
		return fmt.Errorf("failed to write cache metadata file '%s': %v", metaPath, err)
	}

	return nil
}
