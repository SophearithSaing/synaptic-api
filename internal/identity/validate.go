package identity

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Message templates follow the fixture-pinned class-validator formats.
const (
	msgString   = "%s must be a string"
	msgNotEmpty = "%s should not be empty"
	msgMin      = "%s must be longer than or equal to %d characters"
	msgMax      = "%s must be shorter than or equal to %d characters"
	msgEmail    = "%s must be an email"
	msgMatches  = "%s must match %s regular expression"
	msgUnknown  = "property %s should not exist"
)

// Pattern sources appear verbatim in the pinned Matches messages.
const (
	passwordPatternSrc = "/^(?=.*[a-z])(?=.*[A-Z])(?=.*\\d).+$/"
	usernamePatternSrc = "/^[a-zA-Z0-9_.-]+$/"
	identifierPatternS = "/^\\S+$/"
)

var (
	usernamePattern   = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
	identifierPattern = regexp.MustCompile(`^\S+$`)
	authValidator     = newAuthValidator()
)

// registerRequest is the validated registration input.
type registerRequest struct {
	Username string
	Email    string
	Password string
}

// loginRequest is the validated login input.
type loginRequest struct {
	Identifier string
	Password   string
}

// newAuthValidator configures reusable application-specific rules on
// go-playground/validator. A registration failure is a programming
// error and therefore fails during package initialization.
func newAuthValidator() *validator.Validate {
	validate := validator.New(validator.WithRequiredStructEnabled())
	mustRegisterValidation(validate, "username", func(fl validator.FieldLevel) bool {
		return usernamePattern.MatchString(fl.Field().String())
	})
	mustRegisterValidation(
		validate,
		"password_complex",
		func(fl validator.FieldLevel) bool {
			return passwordComplex(fl.Field().String())
		},
	)
	mustRegisterValidation(
		validate,
		"bcrypt_password",
		func(fl validator.FieldLevel) bool {
			return len([]byte(fl.Field().String())) <= 72
		},
	)
	mustRegisterValidation(validate, "no_space", func(fl validator.FieldLevel) bool {
		return identifierPattern.MatchString(fl.Field().String())
	})

	return validate
}

// mustRegisterValidation registers one custom validator or panics when
// its static configuration is invalid.
func mustRegisterValidation(
	validate *validator.Validate,
	tag string,
	rule validator.Func,
) {
	if err := validate.RegisterValidation(tag, rule); err != nil {
		panic(fmt.Sprintf("register %s validation: %v", tag, err))
	}
}

// passwordComplex reports whether the password contains an ASCII
// lowercase letter, uppercase letter, and digit.
func passwordComplex(value string) bool {
	var lower, upper, digit bool
	for _, char := range value {
		switch {
		case 'a' <= char && char <= 'z':
			lower = true
		case 'A' <= char && char <= 'Z':
			upper = true
		case '0' <= char && char <= '9':
			digit = true
		}
	}

	return lower && upper && digit
}

// decodeRegisterRequest transforms and validates registration fields.
// Unknown properties are reported first, followed by field violations
// in request declaration order.
func decodeRegisterRequest(
	fields map[string]json.RawMessage,
) (registerRequest, []string) {
	messages := unknownFieldMessages(
		fields, "username", "email", "password",
	)

	username, usernameIsString := decodeStringField(
		fields["username"], strings.TrimSpace,
	)
	email, emailIsString := decodeStringField(
		fields["email"], trimLower,
	)
	password, _ := decodeStringField(fields["password"], nil)

	if !usernameIsString {
		messages = append(messages, fmt.Sprintf(msgString, "username"))
	} else {
		messages = appendUsernameMessages(messages, username)
	}

	if !emailIsString {
		messages = append(messages, fmt.Sprintf(msgString, "email"))
	} else {
		messages = appendEmailMessages(messages, email)
	}

	messages = appendPasswordMessages(messages, password, true)

	return registerRequest{
		Username: username,
		Email:    email,
		Password: password,
	}, messages
}

// decodeLoginRequest transforms and validates login fields. Unknown
// properties are reported before field violations.
func decodeLoginRequest(
	fields map[string]json.RawMessage,
) (loginRequest, []string) {
	messages := unknownFieldMessages(fields, "identifier", "password")

	identifier, _ := decodeStringField(
		fields["identifier"], strings.TrimSpace,
	)
	password, _ := decodeStringField(fields["password"], nil)

	messages = appendIdentifierMessages(messages, identifier)
	messages = appendPasswordMessages(messages, password, false)

	return loginRequest{
		Identifier: identifier,
		Password:   password,
	}, messages
}

// unknownFieldMessages returns stable forbid-non-whitelisted messages.
func unknownFieldMessages(
	fields map[string]json.RawMessage,
	allowed ...string,
) []string {
	allowedFields := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		allowedFields[name] = struct{}{}
	}

	var messages []string
	for name := range fields {
		if _, ok := allowedFields[name]; !ok {
			messages = append(messages, fmt.Sprintf(msgUnknown, name))
		}
	}
	sort.Strings(messages)

	return messages
}

// decodeStringField decodes one optional JSON string and transforms it.
// The boolean distinguishes valid strings from missing and non-string
// values where the legacy contract needs an IsString message.
func decodeStringField(
	raw json.RawMessage,
	transform func(string) string,
) (string, bool) {
	if raw == nil {
		return "", false
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	if transform != nil {
		value = transform(value)
	}

	return value, true
}

// appendUsernameMessages validates all username rules independently so
// the response retains the legacy all-errors contract.
func appendUsernameMessages(messages []string, value string) []string {
	messages = appendRuleMessage(
		messages, value, "required",
		fmt.Sprintf(msgNotEmpty, "username"),
	)
	messages = appendRuleMessage(
		messages, value, "min=3",
		fmt.Sprintf(msgMin, "username", 3),
	)
	messages = appendRuleMessage(
		messages, value, "max=32",
		fmt.Sprintf(msgMax, "username", 32),
	)

	return appendRuleMessage(
		messages, value, "username",
		fmt.Sprintf(msgMatches, "username", usernamePatternSrc),
	)
}

// appendEmailMessages validates all email rules independently.
func appendEmailMessages(messages []string, value string) []string {
	messages = appendRuleMessage(
		messages, value, "required",
		fmt.Sprintf(msgNotEmpty, "email"),
	)
	messages = appendRuleMessage(
		messages, value, "max=254",
		fmt.Sprintf(msgMax, "email", 254),
	)

	return appendRuleMessage(
		messages, value, "email",
		fmt.Sprintf(msgEmail, "email"),
	)
}

// appendIdentifierMessages validates all login identifier rules.
func appendIdentifierMessages(messages []string, value string) []string {
	messages = appendRuleMessage(
		messages, value, "required",
		fmt.Sprintf(msgNotEmpty, "identifier"),
	)
	messages = appendRuleMessage(
		messages, value, "max=254",
		fmt.Sprintf(msgMax, "identifier", 254),
	)

	return appendRuleMessage(
		messages, value, "no_space",
		fmt.Sprintf(msgMatches, "identifier", identifierPatternS),
	)
}

// appendPasswordMessages validates common password rules and optionally
// applies registration complexity.
func appendPasswordMessages(
	messages []string,
	value string,
	complex bool,
) []string {
	messages = appendRuleMessage(
		messages, value, "required",
		fmt.Sprintf(msgNotEmpty, "password"),
	)
	messages = appendRuleMessage(
		messages, value, "min=8",
		fmt.Sprintf(msgMin, "password", 8),
	)
	if ruleFails(value, "max=72") || ruleFails(value, "bcrypt_password") {
		messages = append(messages, fmt.Sprintf(msgMax, "password", 72))
	}
	if !complex {
		return messages
	}

	return appendRuleMessage(
		messages, value, "password_complex",
		fmt.Sprintf(msgMatches, "password", passwordPatternSrc),
	)
}

// appendRuleMessage appends a pinned message when validator rejects a
// value for one rule.
func appendRuleMessage(
	messages []string,
	value string,
	rule string,
	message string,
) []string {
	if ruleFails(value, rule) {
		return append(messages, message)
	}

	return messages
}

// ruleFails reports whether validator rejects one value and rule.
func ruleFails(value string, rule string) bool {
	return authValidator.Var(value, rule) != nil
}

// trimLower is the email input transformer.
func trimLower(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
