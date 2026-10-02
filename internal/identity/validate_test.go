package identity

import (
	"encoding/json"
	"strings"
	"testing"
)

// rawFields parses a test request object.
func rawFields(t *testing.T, body string) map[string]json.RawMessage {
	t.Helper()

	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &fields); err != nil {
		t.Fatalf("invalid test body %s: %v", body, err)
	}

	return fields
}

// registerMessages validates a registration test body.
func registerMessages(t *testing.T, body string) []string {
	t.Helper()

	_, messages := decodeRegisterRequest(rawFields(t, body))

	return messages
}

// loginMessages validates a login test body.
func loginMessages(t *testing.T, body string) []string {
	t.Helper()

	_, messages := decodeLoginRequest(rawFields(t, body))

	return messages
}

// bodyOf builds a raw JSON body for the given values.
func bodyOf(username, email, password string) string {
	return `{"username":` + jsonValue(username) +
		`,"email":` + jsonValue(email) +
		`,"password":` + jsonValue(password) + `}`
}

// jsonValue quotes a string value.
func jsonValue(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return string(encoded)
}

// TestValidateRegisterMessages pins per-property violation messages.
func TestValidateRegisterMessages(t *testing.T) {
	cases := []struct{ body, join string }{
		{
			body: bodyOf("", "a@e.com", "Password1"),
			join: strings.Join([]string{
				"username should not be empty",
				"username must be longer than or equal to 3 characters",
				"username must match /^[a-zA-Z0-9_.-]+$/ regular expression",
			}, "|"),
		},
		{
			body: bodyOf("al", "nope", "Password1"),
			join: strings.Join([]string{
				"username must be longer than or equal to 3 characters",
				"email must be an email",
			}, "|"),
		},
		{
			body: bodyOf("bad name!", "a@e.com", "Password1"),
			join: "username must match /^[a-zA-Z0-9_.-]+$/ regular " +
				"expression",
		},
		{
			body: bodyOf("alice", "a@e.com", "weak"),
			join: strings.Join([]string{
				"password must be longer than or equal to 8 characters",
				"password must match " +
					"/^(?=.*[a-z])(?=.*[A-Z])(?=.*\\d).+$/ regular expression",
			}, "|"),
		},
		{
			body: bodyOf("alice", "a@e.com", "lowercase1"),
			join: "password must match " +
				"/^(?=.*[a-z])(?=.*[A-Z])(?=.*\\d).+$/ regular expression",
		},
	}

	for _, want := range cases {
		messages := registerMessages(t, want.body)
		if got := strings.Join(messages, "|"); got != want.join {
			t.Errorf("body %s: messages %q, want %q", want.body, got, want.join)
		}
	}
}

// TestValidateAcceptsValidInput pins the happy path.
func TestValidateAcceptsValidInput(t *testing.T) {
	request, messages := decodeRegisterRequest(rawFields(
		t, bodyOf("alice", "A@e.com", "Password1"),
	))
	if messages != nil {
		t.Fatalf("messages %v", messages)
	}
	if request.Email != "a@e.com" {
		t.Fatalf("normalized email %q", request.Email)
	}
}

// TestValidateLoginMessages pins the login schema.
func TestValidateLoginMessages(t *testing.T) {
	messages := loginMessages(
		t, `{"identifier":"no space here","password":"Password1"}`,
	)
	if len(messages) != 1 ||
		messages[0] != "identifier must match /^\\S+$/ regular expression" {
		t.Fatalf("messages %v", messages)
	}

	if messages := loginMessages(
		t, `{"identifier":"s","password":"Password1"}`,
	); messages != nil {
		t.Fatalf("messages %v", messages)
	}
}

// TestValidateUnknownProperties pins stable unknown-field messages.
func TestValidateUnknownProperties(t *testing.T) {
	messages := registerMessages(
		t,
		`{"username":"alice","email":"a@e.com","password":"Password1",`+
			`"scope":"all","role":"admin"}`,
	)

	want := []string{
		"property role should not exist",
		"property scope should not exist",
	}
	if strings.Join(messages, "|") != strings.Join(want, "|") {
		t.Fatalf("messages %v", messages)
	}
}

// TestValidateTrimsBeforeCheck pins the transformer order.
func TestValidateTrimsBeforeCheck(t *testing.T) {
	request, messages := decodeLoginRequest(rawFields(
		t, `{"identifier":" student ","password":"Password1"}`,
	))
	if messages != nil {
		t.Fatalf("messages %v", messages)
	}
	if request.Identifier != "student" {
		t.Fatalf("identifier %q", request.Identifier)
	}
}

// TestValidateTypeFailures pins non-string behavior. String-typed
// properties report the type message; other fields validate as empty.
func TestValidateTypeFailures(t *testing.T) {
	messages := registerMessages(
		t, `{"username":5,"email":7,"password":9}`,
	)
	if messages == nil || messages[0] != "username must be a string" {
		t.Fatalf("messages %v", messages)
	}

	messages = registerMessages(t, `{}`)
	if messages == nil || messages[0] != "username must be a string" {
		t.Fatalf("messages %v", messages)
	}
}

// TestValidateCountsCharactersInsteadOfBytes prevents multibyte input
// from bypassing the password character minimum.
func TestValidateCountsCharactersInsteadOfBytes(t *testing.T) {
	messages := registerMessages(
		t, bodyOf("alice", "a@e.com", "Aa1😊😊"),
	)
	if len(messages) != 1 ||
		messages[0] != "password must be longer than or equal to 8 characters" {
		t.Fatalf("messages %v", messages)
	}
}

// TestValidateRespectsBcryptByteLimit keeps valid character counts from
// reaching bcrypt with an unsupported byte length.
func TestValidateRespectsBcryptByteLimit(t *testing.T) {
	password := "Aa1" + strings.Repeat("😊", 18)
	messages := registerMessages(
		t, bodyOf("alice", "a@e.com", password),
	)
	if len(messages) != 1 ||
		messages[0] != "password must be shorter than or equal to 72 characters" {
		t.Fatalf("messages %v", messages)
	}
}

// TestValidateRejectsMalformedEmail exercises validator's email rule
// beyond the former permissive regular expression.
func TestValidateRejectsMalformedEmail(t *testing.T) {
	messages := registerMessages(
		t, bodyOf("alice", "a..b@example.com", "Password1"),
	)
	if len(messages) != 1 || messages[0] != "email must be an email" {
		t.Fatalf("messages %v", messages)
	}
}
