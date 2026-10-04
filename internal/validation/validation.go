// Package validation configures reusable request validation.
package validation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	translations "github.com/go-playground/validator/v10/translations/en"
)

// Rule defines a custom validation tag. Message uses {0} for the JSON
// field name.
type Rule struct {
	Tag      string
	Validate validator.Func
	Message  string
}

// Validator validates request structs and translates their errors.
type Validator struct {
	engine     *validator.Validate
	translator ut.Translator
}

// New builds a Validator.
func New(rules ...Rule) (*Validator, error) {
	engine := validator.New(validator.WithRequiredStructEnabled())
	engine.RegisterTagNameFunc(jsonFieldName)

	locale := en.New()
	translator, found := ut.New(locale, locale).GetTranslator("en")
	if !found {
		return nil, errors.New("English validator translator is unavailable")
	}
	if err := translations.RegisterDefaultTranslations(
		engine,
		translator,
	); err != nil {
		return nil, fmt.Errorf("register validator translations: %w", err)
	}

	for _, rule := range rules {
		if err := registerRule(engine, translator, rule); err != nil {
			return nil, err
		}
	}

	return &Validator{engine: engine, translator: translator}, nil
}

// Validate returns translated validation messages for value.
func (v *Validator) Validate(value any) []string {
	err := v.engine.Struct(value)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		panic(fmt.Sprintf("validate %T: %v", value, err))
	}

	messages := make([]string, 0, len(validationErrors))
	for _, fieldError := range validationErrors {
		messages = append(messages, fieldError.Translate(v.translator))
	}

	return messages
}

// jsonFieldName returns a field's JSON name for translated messages.
func jsonFieldName(field reflect.StructField) string {
	name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	if name == "" || name == "-" {
		return field.Name
	}

	return name
}

// registerRule registers a custom validation and translation.
func registerRule(
	engine *validator.Validate,
	translator ut.Translator,
	rule Rule,
) error {
	if err := engine.RegisterValidation(rule.Tag, rule.Validate); err != nil {
		return fmt.Errorf("register %s validation: %w", rule.Tag, err)
	}

	err := engine.RegisterTranslation(
		rule.Tag,
		translator,
		func(translator ut.Translator) error {
			return translator.Add(rule.Tag, rule.Message, true)
		},
		func(
			translator ut.Translator,
			fieldError validator.FieldError,
		) string {
			message, err := translator.T(rule.Tag, fieldError.Field())
			if err != nil {
				return fieldError.Error()
			}

			return message
		},
	)
	if err != nil {
		return fmt.Errorf("register %s translation: %w", rule.Tag, err)
	}

	return nil
}
