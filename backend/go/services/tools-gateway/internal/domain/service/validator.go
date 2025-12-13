package service

import (
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

// SchemaValidator validates JSON data against JSON schemas
type SchemaValidator interface {
	// ValidateInput validates input data against a schema
	ValidateInput(input interface{}, schema json.RawMessage) error

	// ValidateOutput validates output data against a schema
	ValidateOutput(output interface{}, schema json.RawMessage) error

	// IsValidSchema checks if a JSON schema is valid
	IsValidSchema(schema json.RawMessage) error
}

// schemaValidatorImpl implements SchemaValidator
type schemaValidatorImpl struct{}

// NewSchemaValidator creates a new SchemaValidator
func NewSchemaValidator() SchemaValidator {
	return &schemaValidatorImpl{}
}

// ValidateInput validates input data against a schema
func (v *schemaValidatorImpl) ValidateInput(input interface{}, schema json.RawMessage) error {
	return v.validate(input, schema, "input")
}

// ValidateOutput validates output data against a schema
func (v *schemaValidatorImpl) ValidateOutput(output interface{}, schema json.RawMessage) error {
	return v.validate(output, schema, "output")
}

// validate performs the actual validation
func (v *schemaValidatorImpl) validate(data interface{}, schemaJSON json.RawMessage, dataType string) error {
	// Parse schema
	schemaLoader := gojsonschema.NewBytesLoader(schemaJSON)

	// Convert data to JSON if needed
	var dataLoader gojsonschema.JSONLoader
	switch d := data.(type) {
	case []byte:
		dataLoader = gojsonschema.NewBytesLoader(d)
	case json.RawMessage:
		dataLoader = gojsonschema.NewBytesLoader([]byte(d))
	case string:
		dataLoader = gojsonschema.NewStringLoader(d)
	default:
		// Convert to JSON bytes
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal %s data: %w", dataType, err)
		}
		dataLoader = gojsonschema.NewBytesLoader(jsonBytes)
	}

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, dataLoader)
	if err != nil {
		return fmt.Errorf("validation error for %s: %w", dataType, err)
	}

	if !result.Valid() {
		return v.formatValidationErrors(result, dataType)
	}

	return nil
}

// IsValidSchema checks if a JSON schema is valid
func (v *schemaValidatorImpl) IsValidSchema(schema json.RawMessage) error {
	schemaLoader := gojsonschema.NewBytesLoader(schema)

	// Try to compile the schema
	_, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return fmt.Errorf("invalid JSON schema: %w", err)
	}

	return nil
}

// formatValidationErrors formats validation errors into a readable message
func (v *schemaValidatorImpl) formatValidationErrors(result *gojsonschema.Result, dataType string) error {
	errors := result.Errors()
	if len(errors) == 0 {
		return fmt.Errorf("%s validation failed", dataType)
	}

	errMsg := fmt.Sprintf("%s validation failed with %d error(s):", dataType, len(errors))
	for _, err := range errors {
		errMsg += fmt.Sprintf("\n  - %s: %s", err.Field(), err.Description())
	}

	return fmt.Errorf("%s", errMsg)
}

// ValidationError represents a validation error
type ValidationError struct {
	Field       string `json:"field"`
	Description string `json:"description"`
	Value       string `json:"value,omitempty"`
}

// FormatValidationResult formats a validation result into structured errors
func FormatValidationResult(result *gojsonschema.Result) []ValidationError {
	errors := result.Errors()
	validationErrors := make([]ValidationError, len(errors))

	for i, err := range errors {
		validationErrors[i] = ValidationError{
			Field:       err.Field(),
			Description: err.Description(),
			Value:       fmt.Sprintf("%v", err.Value()),
		}
	}

	return validationErrors
}
