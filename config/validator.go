package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
)

// validate is the global validator instance.
var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom validators
	if err := validate.RegisterValidation("env", validateEnvironment); err != nil {
		panic(fmt.Sprintf("failed to register env validator: %v", err))
	}
	if err := validate.RegisterValidation("file_exists", validateFileExists); err != nil {
		panic(fmt.Sprintf("failed to register file_exists validator: %v", err))
	}
	if err := validate.RegisterValidation("dir_exists", validateDirExists); err != nil {
		panic(fmt.Sprintf("failed to register dir_exists validator: %v", err))
	}
	if err := validate.RegisterValidation("host", validateHost); err != nil {
		panic(fmt.Sprintf("failed to register host validator: %v", err))
	}
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

// validateFileExists validates that a file path exists and is a regular file.
// Empty string is considered valid (optional file path).
func validateFileExists(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if path == "" {
		return true // Empty path is valid (optional)
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// validateDirExists validates that a directory path exists.
// Empty string is considered valid (optional directory path).
func validateDirExists(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if path == "" {
		return true // Empty path is valid (optional)
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// validateHost validates a host string (hostname or IP address).
// Empty string is considered valid (optional host).
func validateHost(fl validator.FieldLevel) bool {
	host := fl.Field().String()
	if host == "" {
		return true // Empty host is valid (optional)
	}

	// Simple validation: hostname should not contain spaces or special chars
	// This is a basic check; more sophisticated validation could be added
	if strings.Contains(host, " ") || strings.Contains(host, "\t") {
		return false
	}

	// Check for valid characters in hostname
	for _, r := range host {
		if !isValidHostChar(r) {
			return false
		}
	}

	return true
}

// isValidHostChar checks if a character is valid in a hostname.
func isValidHostChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '.' || r == ':' || r == '_' // Allow colon for IPv6, underscore for some cases
}
