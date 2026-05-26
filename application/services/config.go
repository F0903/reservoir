package services

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"reservoir/config"
	"reservoir/utils/assertedpath"
)

type ConfigUpdateStatus int

const (
	ConfigUpdateSuccess ConfigUpdateStatus = iota
	ConfigUpdateRestartRequired
)

type ConfigService struct {
	mu              sync.RWMutex
	cfg             *config.Config
	restartRequired bool
}

func (s *ConfigService) Snapshot() (*config.Config, error) {
	if s == nil || s.cfg == nil {
		return nil, ErrConfigUnavailable
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg.Clone()
}

func (s *ConfigService) UpdatePartial(updates map[string]any) (ConfigUpdateStatus, error) {
	if s == nil || s.cfg == nil {
		return ConfigUpdateSuccess, ErrConfigUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	update, err := config.PreparePartialUpdate(s.cfg, updates)
	if err != nil {
		if errors.Is(err, config.ErrValidationFailed) {
			return ConfigUpdateSuccess, fmt.Errorf("%w: %v", ErrConfigValidationFailed, err)
		}
		return ConfigUpdateSuccess, err
	}

	if err := update.Persist(); err != nil {
		return ConfigUpdateSuccess, err
	}

	result, err := update.Commit()
	if err != nil {
		if errors.Is(err, config.ErrValidationFailed) {
			return ConfigUpdateSuccess, fmt.Errorf("%w: %v", ErrConfigValidationFailed, err)
		}
		return ConfigUpdateSuccess, err
	}
	if result.RestartRequired {
		s.restartRequired = true
	}

	if s.restartRequired {
		return ConfigUpdateRestartRequired, nil
	}
	return ConfigUpdateSuccess, nil
}

func (s *ConfigService) RestartRequired() bool {
	if s == nil {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.restartRequired
}

func (s *ConfigService) LogFilePath() string {
	if s == nil || s.cfg == nil {
		return ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.logFilePathLocked()
}

func (s *ConfigService) logFilePathLocked() string {
	return s.cfg.Logging.File.Read()
}

func (s *ConfigService) OpenLogFileRead() (*os.File, error) {
	if s == nil || s.cfg == nil {
		return nil, ErrConfigUnavailable
	}

	s.mu.RLock()
	logFilePath := s.logFilePathLocked()
	s.mu.RUnlock()

	if logFilePath == "" {
		return nil, ErrNoLogFile
	}

	assertedPath, err := assertedpath.TryAssert(logFilePath)
	if err != nil {
		return nil, err
	}

	logFile, err := os.OpenFile(assertedPath.Path, os.O_RDONLY, 0444)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return logFile, nil
}
