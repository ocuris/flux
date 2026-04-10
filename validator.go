package flux

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Validator is the interface for struct validation.
type Validator interface {
	Validate(i interface{}) error
}

// validator is the built-in struct tag validator.
// The emailRegex is shared from validation.go (emailRegexpV).
type validator struct{}

// NewValidator returns a new validator instance.
func NewValidator() *validator {
	return &validator{}
}

// Validate inspects every field of s that carries a "validate" struct tag
// and applies the declared rules in order. Returns the first error found.
func (v *validator) Validate(s interface{}) error {
	val := reflect.ValueOf(s)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		tag := fieldType.Tag.Get("validate")
		if tag == "" {
			continue
		}

		for _, rule := range strings.Split(tag, ",") {
			rule = strings.TrimSpace(rule)
			if err := v.validateRule(field, fieldType.Name, rule); err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *validator) validateRule(field reflect.Value, fieldName, rule string) error {
	switch {
	case rule == "required":
		if isZeroValue(field) {
			return fmt.Errorf("field '%s' is required", fieldName)
		}

	case rule == "email":
		if field.Kind() == reflect.String {
			email := field.String()
			// Reuse the pre-compiled regex from validation.go
			if email != "" && !emailRegexpV.MatchString(email) {
				return fmt.Errorf("field '%s' must be a valid email", fieldName)
			}
		}

	case strings.HasPrefix(rule, "min="):
		min, err := strconv.Atoi(strings.TrimPrefix(rule, "min="))
		if err != nil {
			return fmt.Errorf("invalid min value in tag for '%s'", fieldName)
		}
		switch field.Kind() {
		case reflect.String:
			if len(field.String()) < min {
				return fmt.Errorf("field '%s' must be at least %d characters", fieldName, min)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Int() < int64(min) {
				return fmt.Errorf("field '%s' must be at least %d", fieldName, min)
			}
		}

	case strings.HasPrefix(rule, "max="):
		max, err := strconv.Atoi(strings.TrimPrefix(rule, "max="))
		if err != nil {
			return fmt.Errorf("invalid max value in tag for '%s'", fieldName)
		}
		switch field.Kind() {
		case reflect.String:
			if len(field.String()) > max {
				return fmt.Errorf("field '%s' must be at most %d characters", fieldName, max)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Int() > int64(max) {
				return fmt.Errorf("field '%s' must be at most %d", fieldName, max)
			}
		}

	case strings.HasPrefix(rule, "gte="):
		gte, err := strconv.Atoi(strings.TrimPrefix(rule, "gte="))
		if err != nil {
			return fmt.Errorf("invalid gte value in tag for '%s'", fieldName)
		}
		if field.Kind() >= reflect.Int && field.Kind() <= reflect.Int64 {
			if field.Int() < int64(gte) {
				return fmt.Errorf("field '%s' must be >= %d", fieldName, gte)
			}
		}

	case strings.HasPrefix(rule, "lte="):
		lte, err := strconv.Atoi(strings.TrimPrefix(rule, "lte="))
		if err != nil {
			return fmt.Errorf("invalid lte value in tag for '%s'", fieldName)
		}
		if field.Kind() >= reflect.Int && field.Kind() <= reflect.Int64 {
			if field.Int() > int64(lte) {
				return fmt.Errorf("field '%s' must be <= %d", fieldName, lte)
			}
		}
	}

	return nil
}

// isZeroValue reports whether v is the zero value for its type.
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}
