package identity

import (
	"encoding/json"
	"strings"
	"testing"
)

// runValidate parses a test body and runs validateFields.
func runValidate(
	t *testing.T,
	schema map[string]fieldRules,
	order []string,
	body string,
) []string {
	t.Helper()

	var fields map[string]json.RawMessage
	if json.Unmarshal([]byte(body), &fields) != nil {
		t.Fatalf("invalid test body %s", body)
	}

	return validateFields(fields, schema, order, authTransforms)
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
		messages := runValidate(
			t, registerSchema, registerOrder, want.body,
		)
		if got := strings.Join(messages, "|"); got != want.join {
			t.Errorf("body %s: messages %q, want %q", want.body, got, want.join)
		}
	}
}

// TestValidateAcceptsValidInput pins the happy path.
func TestValidateAcceptsValidInput(t *testing.T) {
	if messages := runValidate(
		t, registerSchema, registerOrder,
		bodyOf("alice", "A@e.com", "Password1"),
	); messages != nil {
		t.Fatalf("messages %v", messages)
	}
}

// TestValidateLoginMessages pins the login schema.
func TestValidateLoginMessages(t *testing.T) {
	messages := runValidate(
		t, loginSchema, loginOrder,
		`{"identifier":"no space here","password":"Password1"}`,
	)
	if len(messages) != 1 ||
		messages[0] != "identifier must match /^\\S+$/ regular expression" {
		t.Fatalf("messages %v", messages)
	}

	if messages := runValidate(
		t, loginSchema, loginOrder,
		`{"identifier":"s","password":"Password1"}`,
	); messages != nil {
		t.Fatalf("messages %v", messages)
	}
}

// TestValidateUnknownProperties pins the unknown-field message.
func TestValidateUnknownProperties(t *testing.T) {
	messages := runValidate(
		t, registerSchema, registerOrder,
		`{"username":"alice","email":"a@e.com","password":"Password1",`+
			`"role":"admin"}`,
	)

	if len(messages) != 1 ||
		messages[0] != "property role should not exist" {
		t.Fatalf("messages %v", messages)
	}
}

// TestValidateTrimsBeforeCheck pins the transformer order.
func TestValidateTrimsBeforeCheck(t *testing.T) {
	messages := runValidate(
		t, loginSchema, loginOrder,
		`{"identifier":" student ","password":"Password1"}`,
	)
	if messages != nil {
		t.Fatalf("messages %v", messages)
	}
}

// TestValidateTypeFailures pins non-string behavior. String-typed
// properties report the type message; others fail like empty strings.
func TestValidateTypeFailures(t *testing.T) {
	messages := runValidate(
		t, registerSchema, registerOrder,
		`{"username":5,"email":7,"password":9}`,
	)

	if messages == nil || messages[0] != "username must be a string" {
		t.Fatalf("messages %v", messages)
	}

	messages = runValidate(t, registerSchema, registerOrder, `{}`)

	if messages == nil || messages[0] != "username must be a string" {
		t.Fatalf("messages %v", messages)
	}
}
