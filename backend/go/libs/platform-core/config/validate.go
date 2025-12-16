package config

import (
	"fmt"
	"reflect"
	"strings"
)

// ValidateRequired checks if the provided config has non-zero values for all required keys.
// Keys should match the mapstructure tags (e.g., "DATABASE_URL") or struct field names.
func ValidateRequired(cfg *Config, required ...string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	tagMap := make(map[string]any)
	val := reflect.ValueOf(cfg).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		value := val.Field(i).Interface()

		tag := field.Tag.Get("mapstructure")
		if tag != "" {
			tagMap[strings.ToUpper(tag)] = value
		}
		tagMap[strings.ToUpper(field.Name)] = value
	}

	var missing []string
	for _, key := range required {
		upper := strings.ToUpper(key)
		if v, ok := tagMap[upper]; !ok || isZero(v) {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required config: %s", strings.Join(missing, ", "))
	}
	return nil
}

func isZero(v any) bool {
	return reflect.ValueOf(v).IsZero()
}
