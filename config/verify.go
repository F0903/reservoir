package config

import (
	"fmt"
	"reflect"
)

func (c *Config) verify() error {
	if err := checkIsSetRecursive(reflect.ValueOf(c)); err != nil {
		return err
	}

	if c.Proxy.Listen.Read() == "" {
		return fmt.Errorf("proxy.listen cannot be empty")
	}

	if c.Webserver.Listen.Read() == "" {
		return fmt.Errorf("webserver.listen cannot be empty")
	}

	if c.Cache.MaxCacheSize.Read().Bytes() <= 0 {
		return fmt.Errorf("cache.max_cache_size must be greater than 0")
	}

	if c.Cache.Memory.MemoryBudgetPercent.Read() < 0 || c.Cache.Memory.MemoryBudgetPercent.Read() > 100 {
		return fmt.Errorf("cache.memory.memory_budget_percent must be between 0 and 100")
	}

	if c.Cache.CleanupInterval.Read().Cast() <= 0 {
		return fmt.Errorf("cache.cleanup_interval must be greater than 0")
	}

	return nil
}

func checkIsSetRecursive(val reflect.Value) error {
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Check if it's a ConfigProp. We need to get the pointer to call methods.
		// If it's a field in a struct we just Elem'd from a pointer, it should be addressable.
		if field.CanAddr() {
			if isSetMethod := field.Addr().MethodByName("IsSet"); isSetMethod.IsValid() {
				returns := isSetMethod.Call(nil)
				isSet := returns[0].Bool()
				if !isSet {
					jsonTag, _ := fieldType.Tag.Lookup("json")
					return fmt.Errorf("missing or uninitialized required configuration property: '%s'", jsonTag)
				}
				continue
			}
		}

		// Recurse into nested structs
		if field.Kind() == reflect.Struct {
			if err := checkIsSetRecursive(field); err != nil {
				return err
			}
		}
	}

	return nil
}
