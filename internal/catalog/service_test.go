package catalog_test

import (
	"context"
	"testing"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
)

func TestCreateQuestionSetsValidatesAllItemsBeforeWriting(t *testing.T) {
	t.Parallel()

	validator, err := catalog.NewAuthoringValidator()
	if err != nil {
		t.Fatal(err)
	}
	repo := &authoringRepository{}
	service := catalog.NewService(repo, validator)
	level := int64(0)
	_, messages, err := service.CreateQuestionSets(context.Background(),
		[]catalog.CreateQuestionSetRequest{
			validQuestionSetRequest(level),
			{Topic: "not-an-object-id", SetType: "regular", Level: &level},
		},
	)
	if err != nil || len(messages) == 0 {
		t.Fatalf("create messages %v, error %v", messages, err)
	}
	if repo.creates != 0 {
		t.Fatalf("create count %d, want 0", repo.creates)
	}
}

func validQuestionSetRequest(level int64) catalog.CreateQuestionSetRequest {
	return catalog.CreateQuestionSetRequest{
		Topic: "665f1e2b9d1a2c3b4d5e0004", SetType: "regular", Level: &level,
		Questions: []catalog.Question{{
			ID: "q", Type: "mcq", Prompt: "Prompt", CorrectOptionID: "a",
			Options:        []catalog.QuestionOption{{ID: "a", Text: "Answer"}},
			TargetConcepts: []string{"concept"},
			Feedback:       catalog.QuestionFeedback{Correct: "Yes", Incorrect: "No"},
			Rubrics: catalog.QuestionRubric{
				KeyPoints: []string{"point"}, Misconceptions: []string{"mistake"},
			},
		}},
	}
}

type authoringRepository struct {
	creates  int
	category *catalog.Category
	topic    *catalog.Topic
	question *catalog.QuestionSet
	updates  []catalog.UpdateQuestionSetRequest
	err      error
}

func (r *authoringRepository) CreateCategory(context.Context, catalog.CreateCategoryRequest) (*catalog.Category, error) {
	return r.category, r.err
}
func (r *authoringRepository) CreateTopic(context.Context, catalog.CreateTopicRequest) (*catalog.Topic, error) {
	return r.topic, r.err
}
func (r *authoringRepository) CreateQuestionSet(context.Context, catalog.CreateQuestionSetRequest) (*catalog.QuestionSet, error) {
	r.creates++
	if r.question != nil || r.err != nil {
		return r.question, r.err
	}
	return &catalog.QuestionSet{}, nil
}
func (r *authoringRepository) UpdateQuestionSet(_ context.Context, _ string, request catalog.UpdateQuestionSetRequest) (*catalog.QuestionSet, error) {
	r.updates = append(r.updates, request)
	return r.question, r.err
}
func (r *authoringRepository) DeleteCategory(context.Context, string) error    { return r.err }
func (r *authoringRepository) DeleteTopic(context.Context, string) error       { return r.err }
func (r *authoringRepository) DeleteQuestionSet(context.Context, string) error { return r.err }
func (r *authoringRepository) SelectQuestionSet(context.Context, string, int64, string) (*catalog.QuestionSet, error) {
	return nil, nil
}
