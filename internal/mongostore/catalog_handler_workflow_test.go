package mongostore_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// catalogAdminRequest sends an authenticated, CSRF-protected catalog request.
func catalogAdminRequest(
	t *testing.T, wiring *catalogWiring, method, path, body string,
) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+wiring.adminToken)
	request.Header.Set("X-CSRF-Token", "csrf")
	request.AddCookie(&http.Cookie{Name: "csrf_token", Value: "csrf"})
	wiring.mux.ServeHTTP(recorder, request)
	return recorder
}

// responseID returns a successful response's public ID.
func responseID(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.ID == "" {
		t.Fatal("response did not contain id")
	}
	return body.ID
}

// TestCatalogAuthoringRoutesWorkflow exercises the full guarded HTTP authoring
// flow against MongoDB, including reference restrictions and cleanup.
func TestCatalogAuthoringRoutesWorkflow(t *testing.T) {
	wiring := newCatalogWiring(t)
	category := catalogAdminRequest(t, wiring, http.MethodPost,
		"/categories/category/create",
		`{"title":"Workflow Category","slug":"workflow-category",`+
			`"description":"Workflow category","icon":"book"}`)
	if category.Code != http.StatusCreated {
		t.Fatalf("create category: %d %s", category.Code, category.Body.String())
	}
	categoryID := responseID(t, category)

	topic := catalogAdminRequest(t, wiring, http.MethodPost, "/topics/create",
		fmt.Sprintf(`{"title":"Workflow Topic","slug":"workflow-topic",`+
			`"description":"Workflow topic","icon":"tag","tags":["tag"],`+
			`"category":"%s"}`, categoryID))
	if topic.Code != http.StatusCreated {
		t.Fatalf("create topic: %d %s", topic.Code, topic.Body.String())
	}
	topicID := responseID(t, topic)

	set := catalogAdminRequest(t, wiring, http.MethodPost, "/questions/create",
		fmt.Sprintf(`[{"topic":"%s","setType":"regular","level":0,`+
			`"questions":[{"id":"q1","type":"mcq","prompt":"Prompt",`+
			`"options":[{"id":"a","text":"Answer"}],`+
			`"correctOptionId":"a","targetConcepts":["concept"],`+
			`"feedback":{"correct":"Yes","incorrect":"No"},`+
			`"rubrics":{"keyPoints":["point"],`+
			`"misconceptions":["mistake"]}}]}]`, topicID))
	if set.Code != http.StatusCreated {
		t.Fatalf("create set: %d %s", set.Code, set.Body.String())
	}
	var sets []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(set.Body.Bytes(), &sets); err != nil || len(sets) != 1 {
		t.Fatalf("created sets %#v, %v", sets, err)
	}

	patch := catalogAdminRequest(t, wiring, http.MethodPatch,
		"/questions/"+sets[0].ID, `{"level":1}`)
	if patch.Code != http.StatusOK {
		t.Fatalf("patch set: %d %s", patch.Code, patch.Body.String())
	}
	restricted := catalogAdminRequest(t, wiring, http.MethodDelete,
		"/topics/"+topicID, "")
	if restricted.Code != http.StatusConflict {
		t.Fatalf("restricted topic delete: %d", restricted.Code)
	}
	for _, path := range []string{
		"/questions/" + sets[0].ID, "/topics/" + topicID,
		"/categories/" + categoryID,
	} {
		deleted := catalogAdminRequest(t, wiring, http.MethodDelete, path, "")
		if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 {
			t.Fatalf("delete %s: %d %s", path, deleted.Code,
				deleted.Body.String())
		}
	}
}
