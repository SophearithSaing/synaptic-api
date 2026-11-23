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

// responseObject decodes a successful object response.
func responseObject(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

// responseArray decodes a successful array response.
func responseArray(t *testing.T, recorder *httptest.ResponseRecorder) []any {
	t.Helper()
	var body []any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

// requireValue asserts an exact JSON object field value.
func requireValue(t *testing.T, body map[string]any, key string, want any) {
	t.Helper()
	if got := body[key]; got != want {
		t.Fatalf("%s: got %#v, want %#v", key, got, want)
	}
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
	categoryBody := responseObject(t, category)
	requireValue(t, categoryBody, "id", categoryID)
	requireValue(t, categoryBody, "title", "Workflow Category")
	requireValue(t, categoryBody, "slug", "workflow-category")
	requireValue(t, categoryBody, "description", "Workflow category")
	requireValue(t, categoryBody, "icon", "book")

	topic := catalogAdminRequest(t, wiring, http.MethodPost, "/topics/create",
		fmt.Sprintf(`{"title":"Workflow Topic","slug":"workflow-topic",`+
			`"description":"Workflow topic","icon":"tag","tags":["tag"],`+
			`"category":"%s"}`, categoryID))
	if topic.Code != http.StatusCreated {
		t.Fatalf("create topic: %d %s", topic.Code, topic.Body.String())
	}
	topicID := responseID(t, topic)
	topicBody := responseObject(t, topic)
	requireValue(t, topicBody, "id", topicID)
	requireValue(t, topicBody, "title", "Workflow Topic")
	requireValue(t, topicBody, "slug", "workflow-topic")
	requireValue(t, topicBody, "description", "Workflow topic")
	requireValue(t, topicBody, "icon", "tag")
	categoryRelation, ok := topicBody["category"].(map[string]any)
	if !ok {
		t.Fatalf("topic category: %#v", topicBody["category"])
	}
	requireValue(t, categoryRelation, "id", categoryID)
	requireValue(t, categoryRelation, "slug", "workflow-category")

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
	sets := responseArray(t, set)
	if len(sets) != 1 {
		t.Fatalf("created sets %#v", sets)
	}
	setBody, ok := sets[0].(map[string]any)
	if !ok {
		t.Fatalf("created set: %#v", sets[0])
	}
	setID, ok := setBody["id"].(string)
	if !ok || setID == "" {
		t.Fatalf("created set id: %#v", setBody["id"])
	}
	requireValue(t, setBody, "topic", topicID)
	requireValue(t, setBody, "setType", "regular")
	requireValue(t, setBody, "level", float64(0))
	if _, ok := setBody["createdAt"].(string); !ok {
		t.Fatalf("created set timestamp: %#v", setBody["createdAt"])
	}
	questions, ok := setBody["questions"].([]any)
	if !ok || len(questions) != 1 {
		t.Fatalf("created set questions: %#v", setBody["questions"])
	}

	patch := catalogAdminRequest(t, wiring, http.MethodPatch,
		"/questions/"+setID, `{"level":1}`)
	if patch.Code != http.StatusOK {
		t.Fatalf("patch set: %d %s", patch.Code, patch.Body.String())
	}
	patched := responseObject(t, patch)
	requireValue(t, patched, "id", setID)
	requireValue(t, patched, "topic", topicID)
	requireValue(t, patched, "setType", "regular")
	requireValue(t, patched, "level", float64(1))
	if got := len(patched["questions"].([]any)); got != 1 {
		t.Fatalf("patched questions: %d", got)
	}
	persisted := catalogAdminRequest(t, wiring, http.MethodGet,
		"/questions/"+setID+"?populateTopic=false", "")
	if persisted.Code != http.StatusOK {
		t.Fatalf("get patched set: %d %s", persisted.Code, persisted.Body.String())
	}
	persistedBody := responseObject(t, persisted)
	requireValue(t, persistedBody, "topic", topicID)
	requireValue(t, persistedBody, "setType", "regular")
	requireValue(t, persistedBody, "level", float64(1))
	if got := len(persistedBody["questions"].([]any)); got != 1 {
		t.Fatalf("persisted questions: %d", got)
	}

	bulk := catalogAdminRequest(t, wiring, http.MethodPatch,
		"/questions/update", fmt.Sprintf(`[{"id":"%s","level":2}]`, setID))
	if bulk.Code != http.StatusOK {
		t.Fatalf("bulk patch set: %d %s", bulk.Code, bulk.Body.String())
	}
	bulkSets := responseArray(t, bulk)
	if len(bulkSets) != 1 {
		t.Fatalf("bulk patched sets: %#v", bulkSets)
	}
	bulkSet, ok := bulkSets[0].(map[string]any)
	if !ok {
		t.Fatalf("bulk patched set: %#v", bulkSets[0])
	}
	requireValue(t, bulkSet, "id", setID)
	requireValue(t, bulkSet, "topic", topicID)
	requireValue(t, bulkSet, "setType", "regular")
	requireValue(t, bulkSet, "level", float64(2))

	for _, duplicate := range []struct {
		path string
		body string
	}{
		{"/categories/category/create", `{"title":"Duplicate",` +
			`"slug":"workflow-category","description":"Duplicate",` +
			`"icon":"book"}`},
		{"/topics/create", fmt.Sprintf(`{"title":"Duplicate",`+
			`"slug":"workflow-topic","description":"Duplicate",`+
			`"icon":"tag","tags":["tag"],"category":"%s"}`, categoryID)},
	} {
		response := catalogAdminRequest(t, wiring, http.MethodPost,
			duplicate.path, duplicate.body)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("duplicate %s: %d %s", duplicate.path, response.Code,
				response.Body.String())
		}
	}

	restricted := catalogAdminRequest(t, wiring, http.MethodDelete,
		"/topics/"+topicID, "")
	if restricted.Code != http.StatusConflict || restricted.Body.String() !=
		`{"message":"Topic is referenced","error":"Conflict","statusCode":409}`+"\n" {
		t.Fatalf("restricted topic delete: %d %s", restricted.Code,
			restricted.Body.String())
	}
	for _, path := range []string{
		"/questions/" + setID, "/topics/" + topicID,
		"/categories/" + categoryID,
	} {
		deleted := catalogAdminRequest(t, wiring, http.MethodDelete, path, "")
		if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 {
			t.Fatalf("delete %s: %d %s", path, deleted.Code,
				deleted.Body.String())
		}
	}
}
