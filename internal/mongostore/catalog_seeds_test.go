package mongostore_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// objectIDOf parses a fixed hex seed id.
func objectIDOf(t *testing.T, hex string) bson.ObjectID {
	t.Helper()

	objectID, err := bson.ObjectIDFromHex(hex)
	if err != nil {
		t.Fatalf("seed id %q: %v", hex, err)
	}

	return objectID
}

// catalogGet calls GET with the bearer token and returns the status
// code and trimmed body.
func catalogGet(
	t *testing.T,
	wiring *catalogWiring,
	path string,
) string {
	t.Helper()

	served := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Authorization", "Bearer "+wiring.token)
	wiring.mux.ServeHTTP(served, request)

	body, err := io.ReadAll(served.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	status := served.Result().Status
	status = strings.TrimSuffix(status, strings.TrimPrefix(status[:3], ""))

	return status[:3] + "|" + strings.TrimSpace(string(body))
}

// assertCatalogBody pins a raw catalog response body and status.
func assertCatalogBody(
	t *testing.T,
	response string,
	status string,
	want string,
) {
	t.Helper()

	if response != status+"|"+want {
		t.Fatalf("got %q, want %q", response, status+"|"+want)
	}
}
