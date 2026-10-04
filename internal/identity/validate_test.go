package identity

import (
	"reflect"
	"strings"
	"testing"
)

func TestValidateRegisterRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		request  registerRequest
		messages []string
	}{
		{
			name: "missing values",
			messages: []string{
				"username is a required field",
				"email is a required field",
				"password is a required field",
			},
		},
		{
			name: "short username and malformed email",
			request: registerRequest{
				Username: "al",
				Email:    "nope",
				Password: "Password1",
			},
			messages: []string{
				"username must be at least 3 characters in length",
				"email must be a valid email address",
			},
		},
		{
			name: "unsupported username characters",
			request: registerRequest{
				Username: "bad name!",
				Email:    "a@example.com",
				Password: "Password1",
			},
			messages: []string{
				"username must contain only letters, numbers, dots, " +
					"underscores, or hyphens",
			},
		},
		{
			name: "short password",
			request: registerRequest{
				Username: "alice",
				Email:    "a@example.com",
				Password: "weak",
			},
			messages: []string{
				"password must be at least 8 characters in length",
			},
		},
		{
			name: "weak password",
			request: registerRequest{
				Username: "alice",
				Email:    "a@example.com",
				Password: "lowercase1",
			},
			messages: []string{
				"password must contain a lowercase letter, uppercase " +
					"letter, and number",
			},
		},
		{
			name: "valid",
			request: registerRequest{
				Username: "alice",
				Email:    "a@example.com",
				Password: "Password1",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			messages := authValidator.Validate(test.request)
			if !reflect.DeepEqual(messages, test.messages) {
				t.Fatalf("messages %v, want %v", messages, test.messages)
			}
		})
	}
}

func TestValidateLoginRequest(t *testing.T) {
	t.Parallel()

	request := loginRequest{
		Identifier: "no space here",
		Password:   "Password1",
	}
	want := []string{"identifier must not contain whitespace"}
	if messages := authValidator.Validate(request); !reflect.DeepEqual(
		messages,
		want,
	) {
		t.Fatalf("messages %v, want %v", messages, want)
	}

	request.Identifier = "s"
	if messages := authValidator.Validate(request); messages != nil {
		t.Fatalf("messages %v", messages)
	}
}

func TestNormalizeRequests(t *testing.T) {
	t.Parallel()

	registration := registerRequest{
		Username: " alice ",
		Email:    " Alice@Example.COM ",
	}
	registration.normalize()
	if registration.Username != "alice" {
		t.Fatalf("username %q", registration.Username)
	}
	if registration.Email != "alice@example.com" {
		t.Fatalf("email %q", registration.Email)
	}

	login := loginRequest{Identifier: " student "}
	login.normalize()
	if login.Identifier != "student" {
		t.Fatalf("identifier %q", login.Identifier)
	}
}

func TestValidateCountsCharacters(t *testing.T) {
	t.Parallel()

	request := registerRequest{
		Username: "alice",
		Email:    "a@example.com",
		Password: "Aa1😊😊",
	}
	want := []string{"password must be at least 8 characters in length"}
	if messages := authValidator.Validate(request); !reflect.DeepEqual(
		messages,
		want,
	) {
		t.Fatalf("messages %v, want %v", messages, want)
	}
}

func TestValidateBcryptByteLimit(t *testing.T) {
	t.Parallel()

	request := registerRequest{
		Username: "alice",
		Email:    "a@example.com",
		Password: "Aa1" + strings.Repeat("😊", 18),
	}
	want := []string{"password must be at most 72 bytes"}
	if messages := authValidator.Validate(request); !reflect.DeepEqual(
		messages,
		want,
	) {
		t.Fatalf("messages %v, want %v", messages, want)
	}
}
