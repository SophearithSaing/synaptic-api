package catalog_test

import (
	"net/http"
	"time"

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
	topic := catalog.Topic{
		ID:          "665f1e2b9d1a2c3b4d5e0004",
		Title:       "Binary Basics",
		Slug:        "binary-basics",
		Description: "Binary numbers and arithmetic.",
		Icon:        "binary",
		Tags:        []string{"binary", "arithmetic"},
		CategoryID:  "665f1e2b9d1a2c3b4d5e0003",
		Category: &catalog.Category{
			ID:          "665f1e2b9d1a2c3b4d5e0003",
			Title:       "Core Concepts",
			Slug:        "core-concepts",
			Description: "Foundational computing theory.",
			Icon:        "cpu",
		},
	}
	repo.seedTopic(topic)

	seedQuestions := []catalog.Question{{
		ID:     "seed-l0-q1",
		Type:   "mcq",
		Prompt: "What is 1 + 1 in binary?",
		Options: []catalog.QuestionOption{
			{ID: "o1", Text: "seed-l0-q1 option one"},
			{ID: "o2", Text: "seed-l0-q1 option two"},
			{ID: "o3", Text: "seed-l0-q1 option three"},
		},
		CorrectOptionID: "o1",
		TargetConcepts:  []string{"binary-addition"},
		Feedback: catalog.QuestionFeedback{
			Correct:   "Correct feedback.",
			Incorrect: "Incorrect feedback.",
		},
		Rubrics: catalog.QuestionRubric{
			KeyPoints:      []string{"Key point."},
			Misconceptions: []string{"Misconception."},
		},
	}}
	seedDate := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	questionSet := catalog.QuestionSet{
		ID:        "665f1e2b9d1a2c3b4d5e0006",
		TopicID:   topic.ID,
		Topic:     &topic,
		SetType:   "regular",
		Level:     0,
		Questions: seedQuestions,
		CreatedAt: seedDate,
		UpdatedAt: seedDate,
	}
	repo.seedQuestionSet(questionSet)
	repo.seedQuestionSetsForSlug(
		"binary-basics", []catalog.QuestionSet{questionSet},
	)

	handler, token := buildCatalog(repo)

	return handler, token
}
