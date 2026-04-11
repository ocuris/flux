package flux

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Validator is the interface for struct validation.
type Validator interface {
	Validate(i interface{}) error
}

type validationRule struct {
	name  string
	param string
}

type fieldInfo struct {
	name  string
	index int
	rules []validationRule
}

type structCache struct {
	fields []fieldInfo
}

// validator is the built-in struct tag validator.
type validator struct {
	cache sync.Map // map[reflect.Type]*structCache
}

// NewValidator returns the default validator instance.
func NewValidator() *validator {
	return defaultValidator
}

var defaultValidator = &validator{}

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
	var info *structCache
	if cached, ok := v.cache.Load(typ); ok {
		info = cached.(*structCache)
	} else {
		info = v.parseStruct(typ)
		v.cache.Store(typ, info)
	}

	for _, f := range info.fields {
		field := val.Field(f.index)
		for _, r := range f.rules {
			if err := v.executeRule(field, f.name, r.name, r.param); err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *validator) parseStruct(typ reflect.Type) *structCache {
	info := &structCache{}
	for i := 0; i < typ.NumField(); i++ {
		fieldType := typ.Field(i)
		tag := fieldType.Tag.Get("validate")
		if tag == "" {
			continue
		}

		fInfo := fieldInfo{
			name:  fieldType.Name,
			index: i,
		}

		for _, rStr := range strings.Split(tag, ",") {
			rStr = strings.TrimSpace(rStr)
			parts := strings.SplitN(rStr, "=", 2)
			ruleName := parts[0]
			param := ""
			if len(parts) > 1 {
				param = parts[1]
			}
			fInfo.rules = append(fInfo.rules, validationRule{name: ruleName, param: param})
		}
		info.fields = append(info.fields, fInfo)
	}
	return info
}

func (v *validator) executeRule(field reflect.Value, fieldName, ruleName, param string) error {
	switch ruleName {
	case "required":
		if isZeroValue(field) {
			return fmt.Errorf("field '%s' is required", fieldName)
		}

	case "email":
		if field.Kind() == reflect.String {
			email := field.String()
			if email != "" && !emailRegexpV.MatchString(email) {
				return fmt.Errorf("field '%s' must be a valid email", fieldName)
			}
		}

	case "min":
		min, _ := strconv.Atoi(param)
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

	case "max":
		max, _ := strconv.Atoi(param)
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

	case "gte":
		gte, _ := strconv.Atoi(param)
		if field.Kind() >= reflect.Int && field.Kind() <= reflect.Int64 {
			if field.Int() < int64(gte) {
				return fmt.Errorf("field '%s' must be >= %d", fieldName, gte)
			}
		}

	case "lte":
		lte, _ := strconv.Atoi(param)
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
