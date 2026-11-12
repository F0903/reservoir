package services

import (
	"errors"
	"fmt"
	"os"
	"strings"
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

// Settings that name a file or directory. Changed over the API, they would let a
// dashboard admin read any file through the log viewer, or make the service write
// and delete files anywhere it can reach. They stay settable in the config file
// and by command-line flag.
var pathSettings = [][]string{
	{"logging", "file"},
	{"cache", "file", "dir"},
	{"database", "file"},
	{"proxy", "ca_cert"},
	{"proxy", "ca_key"},
}

// pathSettingIn returns the first path setting that updates contains, or "".
func pathSettingIn(updates map[string]any) string {
	for _, path := range pathSettings {
		var node any = updates
		for _, key := range path {
			object, _ := node.(map[string]any)
			node = object[key]
		}
		if node != nil {
			return strings.Join(path, ".")
		}
	}
	return ""
}

func (s *ConfigService) UpdatePartial(updates map[string]any) (ConfigUpdateStatus, error) {
	if s == nil || s.cfg == nil {
		return ConfigUpdateSuccess, ErrConfigUnavailable
	}
	if name := pathSettingIn(updates); name != "" {
		return ConfigUpdateSuccess, fmt.Errorf("%w: %s can only be changed in the config file or with a command-line flag", ErrConfigValidationFailed, name)
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
