package catalog_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
	"github.com/SophearithSaing/synaptic-api/internal/web"
)

func TestAuthoringValidatorMatchesFixtureExamples(t *testing.T) {
	t.Parallel()

	validator := newAuthoringValidator(t)
	category := catalog.CreateCategoryRequest{
		Title: "No Icon", Slug: "no-icon", Description: "Missing.",
	}
	if got, want := validator.ValidateCreateCategory(category), []string{
		"icon should not be empty", "icon must be a string",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("category messages %v, want %v", got, want)
	}

	topic := catalog.CreateTopicRequest{
		Title: "Too Many Tags", Slug: "too-many-tags", Description: "Three tags.",
		Icon: "tag", Tags: []string{"a", "b", "c"},
		Category: "665f1e2b9d1a2c3b4d5e0003",
	}
	if got, want := validator.ValidateCreateTopic(topic), []string{
		"tags must contain no more than 2 elements",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("topic messages %v, want %v", got, want)
	}

	set := catalog.CreateQuestionSetRequest{
		Topic: "665f1e2b9d1a2c3b4d5e0004", SetType: "regular", Level: int64Pointer(3),
	}
	if got, want := validator.ValidateCreateQuestionSet(set), []string{
		"questions should not be empty", "questions must be an array",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("question-set messages %v, want %v", got, want)
	}
}

func TestQuestionContentInvariants(t *testing.T) {
	t.Parallel()

	questions := []catalog.Question{
		{ID: "duplicate", Type: "mcq", CorrectOptionID: "missing"},
		{ID: "duplicate", Type: "written", Options: []catalog.QuestionOption{{ID: "a"}}},
	}
	got := catalog.ValidateQuestions(questions)
	want := []string{
		"mcq questions must include options",
		"correctOptionId must identify an option",
		"question IDs must be unique",
		"written questions must not include options or correctOptionId",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("messages %v, want %v", got, want)
	}

	questions = []catalog.Question{{
		ID: "mcq", Type: "mcq", CorrectOptionID: "one",
		Options: []catalog.QuestionOption{{ID: "one"}, {ID: "one"}},
	}}
	if got, want := catalog.ValidateQuestions(questions), []string{
		"option IDs must be unique",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("messages %v, want %v", got, want)
	}
}

func TestWrittenQuestionRequiresSharedContent(t *testing.T) {
	t.Parallel()

	validator := newAuthoringValidator(t)
	request := catalog.CreateQuestionSetRequest{
		Topic: "665f1e2b9d1a2c3b4d5e0004", SetType: "regular", Questions: []catalog.Question{{
			ID: "written", Type: "written",
		}},
	}
	messages := validator.ValidateCreateQuestionSet(request)
	for _, want := range []string{
		"prompt is a required field",
		"targetConcepts is a required field",
		"feedback is a required field",
		"rubrics is a required field",
	} {
		if !contains(messages, want) {
			t.Errorf("messages %v do not contain %q", messages, want)
		}
	}
}

func TestQuestionSetValidatesNestedQuestionFields(t *testing.T) {
	t.Parallel()

	validator := newAuthoringValidator(t)
	level := int64(0)
	request := catalog.CreateQuestionSetRequest{
		Topic: "665f1e2b9d1a2c3b4d5e0004", SetType: "regular", Level: &level,
		Questions: []catalog.Question{
			{
				ID: "mcq", Type: "mcq", Prompt: "Prompt", CorrectOptionID: "a",
				Options:        []catalog.QuestionOption{{ID: "a", Text: "Answer"}},
				TargetConcepts: []string{"concept"},
				Feedback:       catalog.QuestionFeedback{Correct: "Yes", Incorrect: "No"},
				Rubrics: catalog.QuestionRubric{
					KeyPoints: []string{"point"}, Misconceptions: []string{"mistake"},
				},
			},
			{
				ID: "written", Type: "written", Prompt: "Explain.",
				TargetConcepts: []string{"concept"},
				Feedback:       catalog.QuestionFeedback{Correct: "Yes", Incorrect: "No"},
				Rubrics: catalog.QuestionRubric{
					KeyPoints: []string{"point"}, Misconceptions: []string{"mistake"},
				},
			},
		},
	}
	if got := validator.ValidateCreateQuestionSet(request); len(got) != 0 {
		t.Fatalf("valid nested questions messages %v", got)
	}

	request.Questions[0].Options[0].Text = ""
	if got, want := validator.ValidateCreateQuestionSet(request), []string{
		"text is a required field",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("bad option messages %v, want %v", got, want)
	}
}

func TestUpdateRequestPreservesOmittedFields(t *testing.T) {
	t.Parallel()

	var request catalog.UpdateQuestionSetRequest
	if err := json.Unmarshal([]byte(`{"level":0,"questions":[]}`), &request); err != nil {
		t.Fatalf("unmarshal update: %v", err)
	}
	if request.Topic != nil || request.SetType != nil {
		t.Fatal("omitted fields must remain nil")
	}
	if request.Level == nil || *request.Level != 0 {
		t.Fatalf("level %#v, want pointer to zero", request.Level)
	}
	if request.Questions == nil || len(*request.Questions) != 0 {
		t.Fatalf("questions %#v, want pointer to empty slice", request.Questions)
	}

	validator := newAuthoringValidator(t)
	if got, want := validator.ValidateUpdateQuestionSet(request), []string{
		"questions must contain at least 1 item",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("messages %v, want %v", got, want)
	}
}

func TestAuthoringRequestsRejectMalformedAndUnknownInput(t *testing.T) {
	t.Parallel()

	for _, body := range []string{
		`{"title":"x","extra":true}`,
		`{"title":`,
	} {
		request := httptest.NewRequest("POST", "/", nil)
		request.Body = io.NopCloser(strings.NewReader(body))
		var decoded catalog.CreateCategoryRequest
		if err := web.DecodeJSON(httptest.NewRecorder(), request, &decoded); err == nil {
			t.Fatalf("DecodeJSON(%q) succeeded", body)
		}
	}
	questionSet := httptest.NewRequest("POST", "/", strings.NewReader(
		`{"topic":"665f1e2b9d1a2c3b4d5e0004","setType":"regular","level":0,"questions":[{"id":"q","unexpected":true}]}`,
	))
	var decodedSet catalog.CreateQuestionSetRequest
	if err := web.DecodeJSON(httptest.NewRecorder(), questionSet, &decodedSet); err == nil {
		t.Fatal("nested unknown field was accepted")
	}

	request := httptest.NewRequest("POST", "/", strings.NewReader("null"))
	var category catalog.CreateCategoryRequest
	if err := web.DecodeJSON(httptest.NewRecorder(), request, &category); err != nil {
		t.Fatalf("decode null: %v", err)
	}
	if got, want := newAuthoringValidator(t).ValidateCreateCategory(category),
		[]string{
			"title should not be empty", "title must be a string",
			"slug should not be empty", "slug must be a string",
			"description should not be empty", "description must be a string",
			"icon should not be empty", "icon must be a string",
		}; !reflect.DeepEqual(got, want) {
		t.Fatalf("null messages %v, want %v", got, want)
	}
}

func TestCreateQuestionSetRequiresWholeNumberLevel(t *testing.T) {
	t.Parallel()

	validator := newAuthoringValidator(t)
	request := catalog.CreateQuestionSetRequest{
		Topic: "665f1e2b9d1a2c3b4d5e0004", SetType: "regular",
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
	if got, want := validator.ValidateCreateQuestionSet(request), []string{
		"level is a required field",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing level messages %v, want %v", got, want)
	}

	var nullLevel catalog.CreateQuestionSetRequest
	if err := json.Unmarshal([]byte(`{"level":null}`), &nullLevel); err != nil {
		t.Fatalf("unmarshal null level: %v", err)
	}
	if got, want := validator.ValidateCreateQuestionSet(nullLevel), []string{
		"topic is a required field",
		"setType is a required field",
		"level is a required field",
		"questions should not be empty",
		"questions must be an array",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("null level messages %v, want %v", got, want)
	}

	for _, body := range []string{
		`{"level":1.5}`,
		`{"level":"one"}`,
	} {
		var decoded catalog.CreateQuestionSetRequest
		if err := json.Unmarshal([]byte(body), &decoded); err == nil {
			t.Fatalf("unmarshal %s succeeded", body)
		}
	}
}

func TestAuthoringReferencesRequireObjectIDs(t *testing.T) {
	t.Parallel()

	validator := newAuthoringValidator(t)
	if got, want := validator.ValidateCreateTopic(catalog.CreateTopicRequest{
		Title: "Topic", Slug: "topic", Description: "Description", Icon: "icon",
		Tags: []string{"tag"}, Category: "not-an-object-id",
	}), []string{"category must be a valid ObjectID"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("category messages %v, want %v", got, want)
	}

	if got, want := validator.ValidateBulkUpdateQuestionSet(
		catalog.BulkUpdateQuestionSetRequest{ID: "not-an-object-id"},
	), []string{"id must be a valid ObjectID"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("bulk messages %v, want %v", got, want)
	}

	invalidTopic := "not-an-object-id"
	if got, want := validator.ValidateUpdateQuestionSet(
		catalog.UpdateQuestionSetRequest{Topic: &invalidTopic},
	), []string{"topic must be a valid ObjectID"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("topic messages %v, want %v", got, want)
	}
}

func newAuthoringValidator(t *testing.T) *catalog.AuthoringValidator {
	t.Helper()

	validator, err := catalog.NewAuthoringValidator()
	if err != nil {
		t.Fatalf("new authoring validator: %v", err)
	}

	return validator
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}

func int64Pointer(value int64) *int64 {
	return &value
}
