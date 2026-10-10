package catalog_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
)

// TestQuestionSetPopulateDefaultsAndMissingTopic pins populate defaults and
// the null boundary for a dangling topic reference.
func TestQuestionSetPopulateDefaultsAndMissingTopic(t *testing.T) {
	handler, token := seededCatalog()

	unpopulated := catalogGet(t, handler, token,
		"/questions/665f1e2b9d1a2c3b4d5e0006?populateTopic=false")
	var response map[string]any
	if err := json.NewDecoder(unpopulated.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response["topic"] != "665f1e2b9d1a2c3b4d5e0004" {
		t.Fatalf("unpopulated topic %#v", response["topic"])
	}
	if _, found := response["topicId"]; found {
		t.Fatal("response exposed topicId")
	}

	repo := newRepoState()
	repo.seedQuestionSet(catalog.QuestionSet{
		ID: "665f1e2b9d1a2c3b4d5e0006", TopicID: "665f1e2b9d1a2c3b4d5e0004",
		SetType: "regular", Level: 0, Questions: []catalog.Question{},
		CreatedAt: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
	})
	handler, token = buildCatalog(repo)
	missing := catalogGet(t, handler, token,
		"/questions/665f1e2b9d1a2c3b4d5e0006")
	if missing.StatusCode != http.StatusOK {
		t.Fatalf("status %d", missing.StatusCode)
	}
	response = nil
	if err := json.NewDecoder(missing.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response["topic"] != nil {
		t.Fatalf("missing populated topic %#v", response["topic"])
	}
}
