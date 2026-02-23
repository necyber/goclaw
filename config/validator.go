package config

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// validate is the global validator instance.
var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom validators
	validate.RegisterValidation("env", validateEnvironment)
}

// ConfigError represents a validation error for a specific field.
type ConfigError struct {
	Field   string
	Message string
	Value   interface{}
}

func (e ConfigError) Error() string {
	return fmt.Sprintf("%s: %s (got %v)", e.Field, e.Message, e.Value)
}

// ValidationErrors is a collection of config errors.
type ValidationErrors []ConfigError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}

	var sb strings.Builder
	sb.WriteString("configuration validation failed:\n")
	for _, err := range e {
		sb.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
	}
	return sb.String()
}

// ValidateWithDetails performs validation and returns detailed errors.
func ValidateWithDetails(cfg *Config) error {
	if err := validate.Struct(cfg); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var details ValidationErrors
			for _, fe := range validationErrors {
				details = append(details, ConfigError{
					Field:   fe.Namespace(),
					Message: formatValidationError(fe),
					Value:   fe.Value(),
				})
			}
			return details
		}
		return err
	}
	return nil
}

// formatValidationError converts validator.FieldError to a human-readable message.
func formatValidationError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of [%s]", fe.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())
	default:
		return fmt.Sprintf("failed validation: %s", fe.Tag())
	}
}

// validateEnvironment is a custom validator for environment values.
func validateEnvironment(fl validator.FieldLevel) bool {
	env := fl.Field().String()
	validEnvs := []string{"development", "staging", "production"}
	for _, valid := range validEnvs {
		if env == valid {
			return true
		}
	}
	return false
}
