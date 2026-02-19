package config

import (
	"fmt"
	"reflect"
)

func (c *Config) verify() error {
	if err := checkIsSetRecursive(reflect.ValueOf(c)); err != nil {
		return err
	}

	if err := c.Proxy.verify(); err != nil {
		return err
	}

	if err := c.Webserver.verify(); err != nil {
		return err
	}

	if err := c.Cache.verify(); err != nil {
		return err
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
			if prop, ok := field.Addr().Interface().(StagedConfigProp); ok {
				if !prop.IsSet() {
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
