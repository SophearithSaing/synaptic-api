package identity

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"

	"github.com/SophearithSaing/synaptic-api/internal/validation"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

var authValidator *validation.Validator

// init configures identity request validation.
func init() {
	validate, err := validation.New(
		validation.Rule{
			Tag:      "username",
			Validate: validUsername,
			Message: "{0} must contain only letters, numbers, dots, " +
				"underscores, or hyphens",
		},
		validation.Rule{
			Tag:      "password_complex",
			Validate: validPasswordComplexity,
			Message: "{0} must contain a lowercase letter, uppercase letter, " +
				"and number",
		},
		validation.Rule{
			Tag:      "bcrypt_password",
			Validate: validBcryptPassword,
			Message:  "{0} must be at most 72 bytes",
		},
		validation.Rule{
			Tag:      "no_space",
			Validate: validWithoutSpace,
			Message:  "{0} must not contain whitespace",
		},
	)
	if err != nil {
		panic("configure identity validation: " + err.Error())
	}

	authValidator = validate
}

type registerRequest struct {
	Username string `json:"username" validate:"required,min=3,max=32,username"`
	Email    string `json:"email" validate:"required,max=254,email"`
	Password string `json:"password" validate:"required,min=8,max=72,password_complex,bcrypt_password"`
}

type loginRequest struct {
	Identifier string `json:"identifier" validate:"required,max=254,no_space"`
	Password   string `json:"password" validate:"required,min=8,max=72,bcrypt_password"`
}

// normalize applies registration input normalization before validation.
func (r *registerRequest) normalize() {
	r.Username = strings.TrimSpace(r.Username)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
}

// normalize applies login input normalization before validation.
func (r *loginRequest) normalize() {
	r.Identifier = strings.TrimSpace(r.Identifier)
}

// validUsername reports whether a username uses supported characters.
func validUsername(field validator.FieldLevel) bool {
	return usernamePattern.MatchString(field.Field().String())
}

// validPasswordComplexity reports whether a password meets complexity rules.
func validPasswordComplexity(field validator.FieldLevel) bool {
	var lower, upper, digit bool
	for _, char := range field.Field().String() {
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

// validBcryptPassword reports whether bcrypt accepts a password's byte length.
func validBcryptPassword(field validator.FieldLevel) bool {
	return len([]byte(field.Field().String())) <= 72
}

// validWithoutSpace reports whether a value contains no whitespace.
func validWithoutSpace(field validator.FieldLevel) bool {
	return !strings.ContainsFunc(field.Field().String(), unicode.IsSpace)
}
