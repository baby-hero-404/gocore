package stdres

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/go-playground/validator/v10"
)

// ValidationError maps a request binding/validation error to a typed *stderr.Error
// with field-level details. target is the struct that was bound; its json tags name the fields.
func ValidationError(target any, err error) *stderr.Error {
	var causes []stderr.FieldError

	// Case 1: Validator errors
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, fe := range ve {
			field := jsonFieldName(target, fe.Field())
			causes = append(causes, stderr.FieldError{
				Field:   field,
				Message: fmt.Sprintf("%s is invalid for rule %s", field, fe.Tag()),
				Tag:     fe.Tag(),
			})
		}
		return stderr.NewValidation("invalid_request_body").WithKey("invalid_request_body").WithDetails(causes...)
	}

	// Case 2: JSON Unmarshal type error
	var ute *json.UnmarshalTypeError
	if errors.As(err, &ute) {
		causes = append(causes, stderr.FieldError{
			Field:   ute.Field,
			Message: fmt.Sprintf("expected type %s for field %s", ute.Type.String(), ute.Field),
		})
		return stderr.NewBadRequest("invalid_request_format").WithKey("invalid_request_format").WithDetails(causes...)
	}

	// Case 3: Generic binding error
	causes = append(causes, stderr.FieldError{Field: "body", Message: err.Error()})
	return stderr.NewBadRequest("invalid_request_format").WithKey("invalid_request_format").WithDetails(causes...)
}

func jsonFieldName(obj any, fieldName string) string {
	if obj == nil {
		return fieldName
	}
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return fieldName
	}
	if f, ok := t.FieldByName(fieldName); ok {
		if tag := f.Tag.Get("json"); tag != "" {
			name := strings.Split(tag, ",")[0]
			if name != "-" && name != "" {
				return name
			}
		}
	}
	return fieldName
}
