package catalog_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// assertBody pins a raw JSON response body and status.
func assertBody(
	t *testing.T,
	response *http.Response,
	status int,
	want string,
) {
	t.Helper()

	if response.StatusCode != status {
		t.Fatalf("status %d", response.StatusCode)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content type %q", got)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(body)) != want {
		t.Fatalf("body %s, want %s", body, want)
	}
}
