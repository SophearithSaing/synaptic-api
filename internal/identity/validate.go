package identity

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
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
	emailPattern      = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)

// passwordComplex reports whether the password contains a lowercase
// letter, an uppercase letter, and a digit, matching the pinned
// password pattern.
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

// violation appends the formatted message when the check fails.
func violation(
	messages []string,
	failed bool,
	format string,
	args ...any,
) []string {
	if failed {
		return append(messages, fmt.Sprintf(format, args...))
	}

	return messages
}

// fieldRules bundles the class-validator rules of one property in
// decorator order.
type fieldRules struct {
	// prop is the property name used in messages.
	prop string
	// isString marks the IsString decorator: non-string values report
	// the type message instead of the empty-value failures.
	isString bool
	min      int
	max      int
	// check has the pin-equating predicate behind the Matches rule;
	// nil means the property has no Matches decorator.
	check func(string) bool
	// patternSrc is the pinned source text in the Matches message.
	patternSrc string
	// email marks the IsEmail decorator.
	email bool
}

// validate reports every violation for an optional string value. A nil
// value is an undefined property; non-string values fail like empty
// strings except for the IsString type message.
func (rules fieldRules) validate(value *string) []string {
	if value == nil {
		if rules.isString {
			return []string{fmt.Sprintf(msgString, rules.prop)}
		}
		return rules.checkFailures("")
	}

	return rules.checkFailures(*value)
}

// checkFailures validates the string value in decorator order.
func (rules fieldRules) checkFailures(value string) []string {
	var messages []string

	messages = violation(messages, value == "", msgNotEmpty, rules.prop)
	if rules.min > 0 {
		messages = violation(
			messages,
			len(value) < rules.min,
			msgMin, rules.prop, rules.min,
		)
	}
	if rules.max > 0 {
		messages = violation(
			messages,
			len(value) > rules.max,
			msgMax, rules.prop, rules.max,
		)
	}
	if rules.email {
		messages = violation(
			messages,
			!emailPattern.MatchString(value),
			msgEmail, rules.prop,
		)
	}
	if rules.check != nil {
		messages = violation(
			messages,
			!rules.check(value),
			msgMatches, rules.prop, rules.patternSrc,
		)
	}

	return messages
}

// registerSchema mirrors the RegisterDto decorators.
var registerSchema = map[string]fieldRules{
	"username": {
		prop:       "username",
		isString:   true,
		min:        3,
		max:        32,
		check:      usernamePattern.MatchString,
		patternSrc: usernamePatternSrc,
	},
	"email": {
		prop:     "email",
		isString: true,
		max:      254,
		email:    true,
	},
	"password": {
		prop:       "password",
		min:        8,
		max:        72,
		check:      passwordComplex,
		patternSrc: passwordPatternSrc,
	},
}

// loginSchema mirrors the LoginDto decorators.
var loginSchema = map[string]fieldRules{
	"identifier": {
		prop:       "identifier",
		max:        254,
		check:      identifierPattern.MatchString,
		patternSrc: identifierPatternS,
	},
	"password": {
		prop: "password",
		min:  8,
		max:  72,
	},
}

// validateFields validates one decoded body against a schema. Unknown
// properties are reported first (sorted for stable bodies), then
// per-property violations in declaration order. Returns nil when the
// values are valid.
func validateFields(
	fields map[string]json.RawMessage,
	schema map[string]fieldRules,
	order []string,
	transforms map[string]func(string) string,
) []string {
	var disallowed []string
	for name := range fields {
		if _, ok := schema[name]; !ok {
			disallowed = append(disallowed, fmt.Sprintf(msgUnknown, name))
		}
	}
	sort.Strings(disallowed)

	var messages []string
	messages = append(messages, disallowed...)

	for _, name := range order {
		rules, ok := schema[name]
		if !ok {
			continue
		}

		messages = append(
			messages,
			rules.validate(decodeField(fields[name], transforms[name]))...,
		)
	}

	return messages
}

// decodeField reads a JSON string value and applies the input
// transformer. Returns nil when the value is missing or not a string.
func decodeField(raw json.RawMessage, transform func(string) string) *string {
	if raw == nil {
		return nil
	}

	var value string
	if json.Unmarshal(raw, &value) != nil {
		return nil
	}
	if transform != nil {
		value = transform(value)
	}

	return &value
}
