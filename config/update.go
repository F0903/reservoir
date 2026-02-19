package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
)

type UpdateStatus int

const (
	UpdateStatusFailed UpdateStatus = iota
	UpdateStatusSuccess
	UpdateStatusRestartRequired
)

type stagedProp interface {
	CommitStaged()
}

type StagedConfigProp interface {
	stagedProp
	UnmarshalJSONStaged(data []byte) error
	IsSet() bool
}

// Dynamically sets the properties of a struct based on the provided map.
// Supports nested structs.
func setPropsFromMapRecursive(val reflect.Value, updates map[string]any) (stagedProps []stagedProp, err error) {
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

			jsonTag, _ := field.Tag.Lookup("json")
			if jsonTag != key {
				continue
			}

			found = true
			if fieldVal.Kind() == reflect.Struct {
				// If the value is a map, it's a nested update
				if nestedUpdates, ok := value.(map[string]any); ok {
					nestedStaged, err := setPropsFromMapRecursive(fieldVal.Addr(), nestedUpdates)
					if err != nil {
						return nil, err
					}
					stagedProps = append(stagedProps, nestedStaged...)
					break
				}

				// Check if it's a ConfigProp
				if fieldVal.CanAddr() {
					fieldAddr := fieldVal.Addr()
					if prop, ok := fieldAddr.Interface().(StagedConfigProp); ok {
						valueBytes, err := json.Marshal(value)
						if err != nil {
							return nil, err
						}

						if err := prop.UnmarshalJSONStaged(valueBytes); err != nil {
							return nil, err
						}

						stagedProps = append(stagedProps, prop)
						break
					}
				}
			}
			break
		}
		if !found {
			slog.Warn("Config property not found", "key", key)
		}
	}

	return stagedProps, nil
}

func setPropsFromMap(cfg *Config, updates map[string]any) (stagedProps []stagedProp, err error) {
	return setPropsFromMapRecursive(reflect.ValueOf(cfg), updates)
}

func UpdatePartialFromConfig(cfg *Config, updates map[string]any) (UpdateStatus, error) {
	slog.Info("Updating config with partial JSON", "updates", updates)

	if updates == nil {
		slog.Error("UpdatePartialFromConfig called with nil updates")
		return UpdateStatusFailed, nil
	}

	slog.Debug("Setting properties from JSON map...", "updates", updates)
	stagedProps, err := setPropsFromMapRecursive(reflect.ValueOf(cfg), updates)
	if err != nil {
		slog.Error("Failed to set properties from map", "error", err)
		return UpdateStatusFailed, fmt.Errorf("%w: %v", ErrUpdateFailed, err)
	}

	slog.Info("Committing updated properties...", "staged_count", len(stagedProps))
	for _, prop := range stagedProps {
		slog.Debug("Committing property...", "prop", prop)
		prop.CommitStaged()
	}

	if err := cfg.verify(); err != nil {
		slog.Error("Updated config failed verification", "error", err)
		return UpdateStatusFailed, fmt.Errorf("%w: %v", ErrUpdateFailed, err)
	}

	if err := cfg.persist(); err != nil {
		slog.Error("Failed to persist updated config", "error", err)
		return UpdateStatusFailed, fmt.Errorf("%w: %v", ErrUpdateFailed, err)
	}

	status := UpdateStatusSuccess
	if IsRestartNeeded() {
		slog.Info("Restart is required after updating config")
		status = UpdateStatusRestartRequired
	}
	return status, nil
}

// Deprecated: Use UpdatePartialFromConfig instead.
func UpdatePartialFromJSON(updates map[string]any) (UpdateStatus, error) {
	return UpdatePartialFromConfig(Global, updates)
}
