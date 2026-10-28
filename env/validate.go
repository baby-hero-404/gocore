package env

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/baby-hero-404/gocore/stderr"
)

func validateRequired(val reflect.Value, parentPath string, errorsList *[]stderr.FieldError) {
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return
	}

	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		key := fieldTagKey(field)
		if key == "-" {
			continue
		}

		fieldPath := key
		if parentPath != "" {
			fieldPath = parentPath + "." + key
		}

		fieldVal := val.Field(i)
		isRequired := field.Tag.Get("required") == "true" || field.Tag.Get("validate") == "required"

		if isRequired && isZeroValue(fieldVal) {
			*errorsList = append(*errorsList, stderr.FieldError{
				Field:   fieldPath,
				Message: fmt.Sprintf("configuration property %q is required and cannot be empty", fieldPath),
				Tag:     "required",
			})
		}

		// Recurse into nested structs
		if fieldVal.Kind() == reflect.Struct || (fieldVal.Kind() == reflect.Pointer && !fieldVal.IsNil() && fieldVal.Elem().Kind() == reflect.Struct) {
			validateRequired(fieldVal, fieldPath, errorsList)
		}
	}
}

func fieldTagKey(field reflect.StructField) string {
	if tag := field.Tag.Get("mapstructure"); tag != "" {
		return strings.Split(tag, ",")[0]
	}
	if tag := field.Tag.Get("json"); tag != "" {
		return strings.Split(tag, ",")[0]
	}
	return strings.ToLower(field.Name)
}

func isZeroValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.String:
		return strings.TrimSpace(v.String()) == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	default:
		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	}
}
