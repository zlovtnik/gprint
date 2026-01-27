package fp

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Validator is a function that validates a value and returns an error if invalid.
type Validator[T any] func(T) error

// ValidationError represents a validation error with field information.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	msgs := make([]string, len(e))
	for i, err := range e {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "; ")
}

// HasErrors returns true if there are any validation errors.
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// Validate runs multiple validators and collects all errors.
func Validate[T any](value T, validators ...Validator[T]) Result[T] {
	var errors ValidationErrors
	for _, v := range validators {
		if err := v(value); err != nil {
			if ve, ok := err.(ValidationError); ok {
				errors = append(errors, ve)
			} else if ves, ok := err.(ValidationErrors); ok {
				errors = append(errors, ves...)
			} else {
				errors = append(errors, ValidationError{Message: err.Error()})
			}
		}
	}
	if len(errors) > 0 {
		return Failure[T](errors)
	}
	return Success(value)
}

// Required validates that a string is not empty.
func Required(field string) Validator[string] {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return ValidationError{Field: field, Message: "is required"}
		}
		return nil
	}
}

// MinLength validates minimum string length.
func MinLength(field string, min int) Validator[string] {
	return func(s string) error {
		if utf8.RuneCountInString(s) < min {
			return ValidationError{Field: field, Message: fmt.Sprintf("must be at least %d characters", min)}
		}
		return nil
	}
}

// MaxLength validates maximum string length.
func MaxLength(field string, max int) Validator[string] {
	return func(s string) error {
		if utf8.RuneCountInString(s) > max {
			return ValidationError{Field: field, Message: fmt.Sprintf("must be at most %d characters", max)}
		}
		return nil
	}
}

// Pattern validates against a regular expression.
func Pattern(field string, pattern *regexp.Regexp, message string) Validator[string] {
	return func(s string) error {
		if pattern == nil {
			return ValidationError{Field: field, Message: message}
		}
		if !pattern.MatchString(s) {
			return ValidationError{Field: field, Message: message}
		}
		return nil
	}
}

// Email validates email format.
var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func Email(field string) Validator[string] {
	return Pattern(field, emailPattern, "must be a valid email address")
}

// Positive validates that a number is positive.
func Positive[T int | int32 | int64 | float32 | float64](field string) Validator[T] {
	return func(n T) error {
		if n <= 0 {
			return ValidationError{Field: field, Message: "must be positive"}
		}
		return nil
	}
}

// NonNegative validates that a number is non-negative.
func NonNegative[T int | int32 | int64 | float32 | float64](field string) Validator[T] {
	return func(n T) error {
		if n < 0 {
			return ValidationError{Field: field, Message: "must be non-negative"}
		}
		return nil
	}
}

// Range validates that a number is within a range.
func Range[T int | int32 | int64 | float32 | float64](field string, min, max T) Validator[T] {
	return func(n T) error {
		if n < min || n > max {
			return ValidationError{Field: field, Message: fmt.Sprintf("must be between %v and %v", min, max)}
		}
		return nil
	}
}

// OneOf validates that a value is one of the allowed values.
func OneOf[T comparable](field string, allowed ...T) Validator[T] {
	return func(v T) error {
		for _, a := range allowed {
			if v == a {
				return nil
			}
		}
		return ValidationError{Field: field, Message: "is not a valid value"}
	}
}

// Custom creates a custom validator.
func Custom[T any](field string, predicate func(T) bool, message string) Validator[T] {
	return func(v T) error {
		if predicate == nil {
			return ValidationError{Field: field, Message: message}
		}
		if !predicate(v) {
			return ValidationError{Field: field, Message: message}
		}
		return nil
	}
}

// NewError creates a formatted error.
func NewError(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// WrapError wraps an error with additional context.
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
