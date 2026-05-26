package services

import (
	"reservoir/config"
	runtimeMetrics "reservoir/metrics"
)

type CacheStatus struct {
	Type           config.CacheType
	Entries        int
	Bytes          int64
	MaxBytes       int64
	MemoryCapBytes *int64
}

type CacheService struct {
	cfg        *config.Config
	controller CacheController
}

func (s *CacheService) Available() bool {
	return s != nil && s.controller != nil && s.cfg != nil
}

func (s *CacheService) Status() (CacheStatus, error) {
	if !s.Available() {
		return CacheStatus{}, ErrCacheUnavailable
	}

	stats := s.controller.CacheStats()
	cacheType := s.cfg.Cache.Type.Read()
	status := CacheStatus{
		Type:     cacheType,
		Entries:  stats.Entries,
		Bytes:    stats.Bytes,
		MaxBytes: stats.MaxBytes,
	}
	if cacheType == config.CacheTypeMemory || cacheType == config.CacheTypeHybrid {
		memoryCapBytes := stats.MemoryCapBytes
		status.MemoryCapBytes = &memoryCapBytes
	}

	return status, nil
}

func (s *CacheService) Clear() error {
	if !s.Available() {
		return ErrCacheUnavailable
	}
	return s.controller.ClearCache()
}

func (s *CacheService) StorageMetrics() (runtimeMetrics.CacheStorageMetrics, bool) {
	status, err := s.Status()
	if err != nil {
		return runtimeMetrics.CacheStorageMetrics{}, false
	}

	return runtimeMetrics.CacheStorageMetrics{
		Type:           string(status.Type),
		Entries:        status.Entries,
		Bytes:          status.Bytes,
		MaxBytes:       status.MaxBytes,
		MemoryCapBytes: status.MemoryCapBytes,
	}, true
}
