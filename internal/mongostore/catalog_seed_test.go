package mongostore_test

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// seedDate is the shared fixed creation timestamp of the seed data.
var seedDate = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// seedFixturesCatalog inserts the capture-shaped seed data with stored
// field order matching the Mongoose inserts (order-preserving bson.D).
func seedFixturesCatalog(
	t *testing.T,
	ctx context.Context,
	database *mongo.Database,
) {
	t.Helper()

	categoryCore := bson.D{
		{Key: "_id", Value: objectIDOf(t, "5eed00000000000000000011")},
		{Key: "title", Value: "Core Concepts"},
		{Key: "slug", Value: "core-concepts"},
		{Key: "description", Value: "Foundational computing theory."},
		{Key: "icon", Value: "cpu"},
		{Key: "createdAt", Value: seedDate},
		{Key: "updatedAt", Value: seedDate},
		{Key: "__v", Value: 0},
	}
	categoryNetwork := bson.D{
		{Key: "_id", Value: objectIDOf(t, "5eed00000000000000000012")},
		{Key: "title", Value: "Networking"},
		{Key: "slug", Value: "networking"},
		{Key: "description", Value: "Networks and protocols."},
		{Key: "icon", Value: "wifi"},
		{Key: "createdAt", Value: seedDate},
		{Key: "updatedAt", Value: seedDate},
		{Key: "__v", Value: 0},
	}

	categories := database.Collection("categories")
	for _, category := range []bson.D{categoryCore, categoryNetwork} {
		if _, err := categories.InsertOne(ctx, category); err != nil {
			t.Fatalf("seed category: %v", err)
		}
	}

	topicBinary := bson.D{
		{Key: "_id", Value: objectIDOf(t, "5eed00000000000000000021")},
		{Key: "title", Value: "Binary Basics"},
		{Key: "slug", Value: "binary-basics"},
		{Key: "description", Value: "Binary numbers and arithmetic."},
		{Key: "icon", Value: "binary"},
		{Key: "tags", Value: plainArray("binary", "arithmetic")},
		{
			Key:   "category",
			Value: objectIDOf(t, "5eed00000000000000000011"),
		},
		{Key: "createdAt", Value: seedDate},
		{Key: "updatedAt", Value: seedDate},
		{Key: "__v", Value: 0},
	}
	topicEmpty := bson.D{
		{Key: "_id", Value: objectIDOf(t, "5eed00000000000000000022")},
		{Key: "title", Value: "Empty Topic"},
		{Key: "slug", Value: "empty-topic"},
		{Key: "description", Value: "Topic without question sets."},
		{Key: "icon", Value: "empty"},
		{Key: "tags", Value: plainArray("misc")},
		{
			Key:   "category",
			Value: objectIDOf(t, "5eed00000000000000000011"),
		},
		{Key: "createdAt", Value: seedDate},
		{Key: "updatedAt", Value: seedDate},
		{Key: "__v", Value: 0},
	}
	topicLogic := bson.D{
		{Key: "_id", Value: objectIDOf(t, "5eed00000000000000000024")},
		{Key: "title", Value: "Logic Gates"},
		{Key: "slug", Value: "logic-gates"},
		{Key: "description", Value: "Boolean logic and gates."},
		{Key: "icon", Value: "gate"},
		{Key: "tags", Value: plainArray("logic")},
		{
			Key:   "category",
			Value: objectIDOf(t, "5eed00000000000000000011"),
		},
		{Key: "createdAt", Value: seedDate},
		{Key: "updatedAt", Value: seedDate},
		{Key: "__v", Value: 0},
	}
	topics := database.Collection("topics")
	for _, topic := range []bson.D{topicBinary, topicEmpty, topicLogic} {
		if _, err := topics.InsertOne(ctx, topic); err != nil {
			t.Fatalf("seed topic: %v", err)
		}
	}

	binaryID := objectIDOf(t, "5eed00000000000000000021")

	questionSets := database.Collection("questionSets")
	for _, questionSet := range []bson.D{
		regularSet(t, "5eed00000000000000000031", binaryID, 0, []any{
			seedQuestionQ1(), seedQuestionQ2(),
		}),
		regularSet(t, "5eed00000000000000000032", binaryID, 1, []any{
			writtenQuestion(),
		}),
		regularSet(t, "5eed00000000000000000033", binaryID, 4, []any{
			createdQuestionQ4(),
		}),
		regularSet(
			t,
			"5eed00000000000000000034",
			objectIDOf(t, "5eed00000000000000000029"),
			0,
			[]any{},
		),
	} {
		if _, err := questionSets.InsertOne(ctx, questionSet); err != nil {
			t.Fatalf("seed question set: %v", err)
		}
	}
}

// regularSet builds a regular question set with fixed order and dates.
func regularSet(
	t *testing.T, id string, topic bson.ObjectID, level int64, questions []any,
) bson.D {
	t.Helper()

	return bson.D{
		{Key: "_id", Value: objectIDOf(t, id)},
		{Key: "topic", Value: topic},
		{Key: "setType", Value: "regular"},
		{Key: "level", Value: level},
		{Key: "questions", Value: questions},
		{Key: "createdAt", Value: seedDate},
		{Key: "updatedAt", Value: seedDate},
	}
}

// mcqBody builds one MCQ question body in stored order.
func mcqBody(
	id, prompt, correctID, optionOne, optionTwo, optionThree, concept string,
) bson.D {
	return bson.D{
		{Key: "id", Value: id},
		{Key: "type", Value: "mcq"},
		{Key: "prompt", Value: prompt},
		{Key: "options", Value: []any{
			option("o1", optionOne),
			option("o2", optionTwo),
			option("o3", optionThree),
		}},
		{Key: "correctOptionId", Value: correctID},
		{Key: "targetConcepts", Value: plainArray(concept)},
		{Key: "feedback", Value: bson.D{
			{Key: "correct", Value: "Correct feedback."},
			{Key: "incorrect", Value: "Incorrect feedback."},
		}},
		{Key: "rubrics", Value: bson.D{
			{Key: "keyPoints", Value: plainArray("Key point.")},
			{Key: "misconceptions", Value: plainArray("Misconception.")},
		}},
	}
}

// option builds one pinned question option with stored order.
func option(id, label string) bson.D {
	return bson.D{
		{Key: "id", Value: id},
		{Key: "text", Value: label},
	}
}

// plainArray builds a stored string array.
func plainArray(values ...string) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}

	return result
}

// seedQuestionQ1 mirrors the seed level-0 question one.
func seedQuestionQ1() any {
	return mcqBody(
		"seed-l0-q1",
		"What is 1 + 1 in binary?",
		"o1",
		"seed-l0-q1 option one",
		"seed-l0-q1 option two",
		"seed-l0-q1 option three",
		"binary-addition",
	)
}

// seedQuestionQ2 mirrors the seed level-0 question two.
func seedQuestionQ2() any {
	return mcqBody(
		"seed-l0-q2",
		"What is 10 + 1 in binary?",
		"o2",
		"seed-l0-q2 option one",
		"seed-l0-q2 option two",
		"seed-l0-q2 option three",
		"binary-addition",
	)
}

// createdQuestionQ4 mirrors the created CPU question.
func createdQuestionQ4() any {
	return bson.D{
		{Key: "id", Value: "created-q1"},
		{Key: "type", Value: "mcq"},
		{Key: "prompt", Value: "What does CPU stand for?"},
		{Key: "options", Value: []any{
			option("o1", "Central Processing Unit"),
			option("o2", "Computer Personal Unit"),
			option("o3", "Central Program Utility"),
		}},
		{Key: "correctOptionId", Value: "o1"},
		{Key: "targetConcepts", Value: plainArray("hardware-basics")},
		{Key: "feedback", Value: bson.D{
			{Key: "correct", Value: "Correct."},
			{Key: "incorrect", Value: "Incorrect."},
		}},
		{Key: "rubrics", Value: bson.D{
			{Key: "keyPoints", Value: plainArray("Central processing.")},
			{Key: "misconceptions", Value: plainArray("Personal computer.")},
		}},
	}
}

// writtenQuestion mirrors the seed level-1 written question in stored
// order.
func writtenQuestion() any {
	return bson.D{
		{Key: "id", Value: "seed-l1-q1"},
		{Key: "type", Value: "written"},
		{Key: "prompt", Value: "Explain two's complement."},
		{Key: "targetConcepts", Value: plainArray("twos-complement")},
		{Key: "feedback", Value: bson.D{
			{Key: "correct", Value: "Correct feedback."},
			{Key: "incorrect", Value: "Incorrect feedback."},
		}},
		{Key: "rubrics", Value: bson.D{
			{Key: "keyPoints", Value: plainArray("Invert bits.", "Add one.")},
			{Key: "misconceptions", Value: plainArray("Sign bit only.")},
		}},
	}
}
