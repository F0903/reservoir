package services

import (
	"errors"
	"fmt"

	"reservoir/config"
)

type ConfigUpdateStatus int

const (
	ConfigUpdateSuccess ConfigUpdateStatus = iota
	ConfigUpdateRestartRequired
)

type ConfigService struct {
	cfg *config.Config
}

func (s *ConfigService) Snapshot() (*config.Config, error) {
	if s == nil || s.cfg == nil {
		return nil, ErrConfigUnavailable
	}
	return s.cfg, nil
}

func (s *ConfigService) UpdatePartial(updates map[string]any) (ConfigUpdateStatus, error) {
	if s == nil || s.cfg == nil {
		return ConfigUpdateSuccess, ErrConfigUnavailable
	}

	status, err := config.UpdatePartialFromConfig(s.cfg, updates)
	if err != nil {
		if errors.Is(err, config.ErrValidationFailed) {
			return ConfigUpdateSuccess, fmt.Errorf("%w: %v", ErrConfigValidationFailed, err)
		}
		return ConfigUpdateSuccess, err
	}

	switch status {
	case config.UpdateStatusRestartRequired:
		return ConfigUpdateRestartRequired, nil
	default:
		return ConfigUpdateSuccess, nil
	}
}

func (s *ConfigService) RestartRequired() bool {
	return config.IsRestartNeeded()
}

func (s *ConfigService) LogFilePath() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return s.cfg.Logging.File.Read()
}
