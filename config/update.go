package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
)

var (
	ErrUpdateFailed     = errors.New("failed to update config")
	ErrValidationFailed = errors.New("config validation failed")
)

type CommitResult struct {
	RestartRequired bool
}

type stagedProp interface {
	CommitStaged() CommitResult
}

type StagedConfigProp interface {
	stagedProp
	UnmarshalJSONStaged(data []byte) error
	IsSet() bool
}

func cloneConfig(cfg *Config) (*Config, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	var cloned Config
	if err := json.Unmarshal(data, &cloned); err != nil {
		return nil, err
	}
	cloned.setRestartNeededProps()
	cloned.path = cfg.path

	return &cloned, nil
}

func (c *Config) Clone() (*Config, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: config cannot be nil", ErrValidationFailed)
	}
	return cloneConfig(c)
}

func commitStagedProps(stagedProps []stagedProp) CommitResult {
	var result CommitResult
	for _, prop := range stagedProps {
		slog.Debug("Committing property...", "prop", prop)
		propResult := prop.CommitStaged()
		result.RestartRequired = result.RestartRequired || propResult.RestartRequired
	}
	return result
}

type PartialUpdate struct {
	cfg     *Config
	updates map[string]any
	next    *Config
}

func PreparePartialUpdate(cfg *Config, updates map[string]any) (*PartialUpdate, error) {
	slog.Info("Preparing config partial update", "updates", updates)

	if cfg == nil {
		return nil, fmt.Errorf("%w: config cannot be nil", ErrValidationFailed)
	}
	if updates == nil {
		return nil, fmt.Errorf("%w: updates cannot be nil", ErrValidationFailed)
	}

	slog.Debug("Setting properties from JSON map...", "updates", updates)
	candidate, err := cloneConfig(cfg)
	if err != nil {
		slog.Error("Failed to clone config before update", "error", err)
		return nil, fmt.Errorf("%w: %v", ErrUpdateFailed, err)
	}

	candidateStagedProps, err := setPropsFromMap(candidate, updates)
	if err != nil {
		slog.Error("Failed to set properties from map", "error", err)
		if errors.Is(err, ErrValidationFailed) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrUpdateFailed, err)
	}

	commitStagedProps(candidateStagedProps)

	if err := candidate.verify(); err != nil {
		slog.Error("Updated config failed verification", "error", err)
		return nil, fmt.Errorf("%w: %v", ErrValidationFailed, err)
	}

	return &PartialUpdate{
		cfg:     cfg,
		updates: updates,
		next:    candidate,
	}, nil
}

func (u *PartialUpdate) Persist() error {
	if u == nil || u.next == nil {
		return fmt.Errorf("%w: partial update is not prepared", ErrUpdateFailed)
	}
	if err := u.next.persist(); err != nil {
		slog.Error("Failed to persist updated config", "error", err)
		return fmt.Errorf("%w: %v", ErrUpdateFailed, err)
	}
	return nil
}

func (u *PartialUpdate) Commit() (CommitResult, error) {
	if u == nil || u.cfg == nil {
		return CommitResult{}, fmt.Errorf("%w: partial update is not prepared", ErrUpdateFailed)
	}

	stagedProps, err := setPropsFromMap(u.cfg, u.updates)
	if err != nil {
		slog.Error("Failed to apply verified config update", "error", err)
		if errors.Is(err, ErrValidationFailed) {
			return CommitResult{}, err
		}
		return CommitResult{}, fmt.Errorf("%w: %v", ErrUpdateFailed, err)
	}

	slog.Info("Committing updated properties...", "staged_count", len(stagedProps))
	return commitStagedProps(stagedProps), nil
}

// Dynamically sets the properties of a struct based on the provided map.
// Supports nested structs.
func setPropsFromMapRecursive(val reflect.Value, updates map[string]any) (stagedProps []stagedProp, err error) {
	return setPropsFromMapRecursiveAtPath(val, updates, "")
}

func setPropsFromMapRecursiveAtPath(val reflect.Value, updates map[string]any, path string) (stagedProps []stagedProp, err error) {
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	typ := val.Type()
	stagedProps = make([]stagedProp, 0)

	for key, value := range updates {
		found := false
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			fieldVal := val.Field(i)

			jsonName, ok := jsonFieldName(field)
			if !ok || jsonName != key {
				continue
			}

			found = true
			keyPath := joinConfigPath(path, key)
			if fieldVal.Kind() == reflect.Struct {
				// Check if it's a ConfigProp
				if fieldVal.CanAddr() {
					fieldAddr := fieldVal.Addr()
					if prop, ok := fieldAddr.Interface().(StagedConfigProp); ok {
						valueBytes, err := json.Marshal(value)
						if err != nil {
							return nil, err
						}

						if err := prop.UnmarshalJSONStaged(valueBytes); err != nil {
							return nil, fmt.Errorf("%w: invalid value for config key %q: %v", ErrValidationFailed, keyPath, err)
						}

						stagedProps = append(stagedProps, prop)
						break
					}
				}

				// If the value is a map, it's a nested update
				if nestedUpdates, ok := value.(map[string]any); ok {
					nestedStaged, err := setPropsFromMapRecursiveAtPath(fieldVal.Addr(), nestedUpdates, keyPath)
					if err != nil {
						return nil, err
					}
					stagedProps = append(stagedProps, nestedStaged...)
					break
				}

				return nil, fmt.Errorf("%w: config key %q expects an object", ErrValidationFailed, keyPath)
			}

			return nil, fmt.Errorf("%w: unsupported config key %q", ErrValidationFailed, keyPath)
		}
		if !found {
			return nil, fmt.Errorf("%w: unknown config key %q", ErrValidationFailed, joinConfigPath(path, key))
		}
	}

	return stagedProps, nil
}

func setPropsFromMap(cfg *Config, updates map[string]any) (stagedProps []stagedProp, err error) {
	return setPropsFromMapRecursive(reflect.ValueOf(cfg), updates)
}

func jsonFieldName(field reflect.StructField) (string, bool) {
	if field.PkgPath != "" {
		return "", false
	}

	jsonTag, ok := field.Tag.Lookup("json")
	if !ok {
		return "", false
	}

	name, _, _ := strings.Cut(jsonTag, ",")
	if name == "" || name == "-" {
		return "", false
	}

	return name, true
}

func joinConfigPath(prefix string, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}
