package validation_test

import (
	"reflect"
	"testing"

	"github.com/go-playground/validator/v10"

	"github.com/SophearithSaing/synaptic-api/internal/validation"
)

type request struct {
	DisplayName string `json:"displayName" validate:"required,min=3,lowercase"`
}

func TestValidateUsesJSONNamesAndTranslations(t *testing.T) {
	t.Parallel()

	validate, err := validation.New(validation.Rule{
		Tag:      "lowercase",
		Validate: isLowercase,
		Message:  "{0} must be lowercase",
	})
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}

	tests := []struct {
		name     string
		request  request
		messages []string
	}{
		{
			name:     "required",
			messages: []string{"displayName is a required field"},
		},
		{
			name:     "minimum",
			request:  request{DisplayName: "ab"},
			messages: []string{"displayName must be at least 3 characters in length"},
		},
		{
			name:     "custom",
			request:  request{DisplayName: "Alice"},
			messages: []string{"displayName must be lowercase"},
		},
		{
			name:    "valid",
			request: request{DisplayName: "alice"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			messages := validate.Validate(test.request)
			if !reflect.DeepEqual(messages, test.messages) {
				t.Fatalf("messages %v, want %v", messages, test.messages)
			}
		})
	}
}

func isLowercase(field validator.FieldLevel) bool {
	value := field.Field().String()
	for _, char := range value {
		if 'A' <= char && char <= 'Z' {
			return false
		}
	}

	return true
}
