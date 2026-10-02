package catalog_test

import (
	"encoding/json"
	"net/http"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
)

// seededCatalog mirrors the fixture seed data.
func seededCatalog() (http.Handler, string) {
	repo := newRepoState()
	repo.seedCategory(catalog.Category{
		ID:          "665f1e2b9d1a2c3b4d5e0003",
		Title:       "Core Concepts",
		Slug:        "core-concepts",
		Description: "Foundational computing theory.",
		Icon:        "cpu",
	})
	repo.seedCategory(catalog.Category{
		ID:          "665f1e2b9d1a2c3b4d5e0009",
		Title:       "Networking",
		Slug:        "networking",
		Description: "Networks and protocols.",
		Icon:        "wifi",
	})
	repo.seedTopic(catalog.Topic{
		ID:          "665f1e2b9d1a2c3b4d5e0004",
		Title:       "Binary Basics",
		Slug:        "binary-basics",
		Description: "Binary numbers and arithmetic.",
		Icon:        "binary",
		Tags:        []string{"binary", "arithmetic"},
		Category: &catalog.Category{
			ID:          "665f1e2b9d1a2c3b4d5e0003",
			Title:       "Core Concepts",
			Slug:        "core-concepts",
			Description: "Foundational computing theory.",
			Icon:        "cpu",
		},
	})

	seedQuestions := json.RawMessage(`[` +
		`{"id":"seed-l0-q1","type":"mcq","prompt":"What is 1 + 1 in binary?",` +
		`"options":[{"id":"o1","text":"seed-l0-q1 option one"},` +
		`{"id":"o2","text":"seed-l0-q1 option two"},` +
		`{"id":"o3","text":"seed-l0-q1 option three"}],` +
		`"correctOptionId":"o1","targetConcepts":["binary-addition"],` +
		`"feedback":{"correct":"Correct feedback.","incorrect":` +
		`"Incorrect feedback."},"rubrics":{"keyPoints":["Key point."],` +
		`"misconceptions":["Misconception."]}}]`)
	questionSet := catalog.QuestionSet{
		ID: "665f1e2b9d1a2c3b4d5e0006",
		Topic: json.RawMessage(
			`"665f1e2b9d1a2c3b4d5e0004"`,
		),
		SetType:   "regular",
		Level:     0,
		Questions: seedQuestions,
		CreatedAt: "2026-01-01T00:00:00.000Z",
		UpdatedAt: "2026-01-01T00:00:00.000Z",
	}
	repo.seedQuestionSet(questionSet)
	repo.seedQuestionSetsForSlug(
		"binary-basics", []catalog.QuestionSet{questionSet},
	)

	handler, token := buildCatalog(repo)

	return handler, token
}
