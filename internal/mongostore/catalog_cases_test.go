package mongostore_test

import (
	"testing"
)

// TestCatalogRoutesFixtureShapes verifies the six catalog routes
// against the real seeded database.
func TestCatalogRoutesFixtureShapes(t *testing.T) {
	wiring := newCatalogWiring(t)

	// GET /categories/categories.
	assertCatalogBody(t,
		catalogGet(t, wiring, "/categories/categories"),
		"200", categoryCoreAndNetwork)

	// GET /categories/{id}.
	assertCatalogBody(t,
		catalogGet(t, wiring, "/categories/5eed00000000000000000011"),
		"200", categoryCore)
	assertCatalogBody(t,
		catalogGet(t, wiring, "/categories/5eed00000000000000000012"),
		"200", categoryNetwork)
	assertCatalogBody(t,
		catalogGet(t, wiring, "/categories/5eed00000000000000000019"),
		"404", `{"message":"Category not found","error":"Not Found",`+
			`"statusCode":404}`)
	assertCatalogBody(t,
		catalogGet(t, wiring, "/categories/not-an-id"),
		"400", `{"message":"Invalid MongoDB ObjectId",`+
			`"error":"Bad Request","statusCode":400}`)

	// GET /topics (sorted by title).
	topics := "[" + topicNested(
		"5eed00000000000000000021", "Binary Basics", "binary-basics",
		"Binary numbers and arithmetic.", "binary",
		`["binary","arithmetic"]`,
	) + "," + topicNested(
		"5eed00000000000000000022", "Empty Topic", "empty-topic",
		"Topic without question sets.", "empty", `["misc"]`,
	) + "," + topicNested(
		"5eed00000000000000000024", "Logic Gates", "logic-gates",
		"Boolean logic and gates.", "gate", `["logic"]`,
	) + "]"
	assertCatalogBody(t,
		catalogGet(t, wiring, "/topics"),
		"200", topics)

	// GET /topics/{id}.
	assertCatalogBody(t,
		catalogGet(t, wiring, "/topics/5eed00000000000000000021"),
		"200", topicNested(
			"5eed00000000000000000021", "Binary Basics",
			"binary-basics", "Binary numbers and arithmetic.",
			"binary", `["binary","arithmetic"]`,
		))
	assertCatalogBody(t,
		catalogGet(t, wiring, "/topics/5eed00000000000000000019"),
		"404", `{"message":"Topic not found","error":"Not Found",`+
			`"statusCode":404}`)
	assertCatalogBody(t,
		catalogGet(t, wiring, "/topics/not-an-id"),
		"400", `{"message":"Invalid MongoDB ObjectId",`+
			`"error":"Bad Request","statusCode":400}`)
}

// TestCatalogQuestionRoutes pins the question shapes end to end.
func TestCatalogQuestionRoutes(t *testing.T) {
	wiring := newCatalogWiring(t)
	// GET /questions/{id} populates the topic relation by default.
	assertCatalogBody(t,
		catalogGet(t, wiring, "/questions/5eed00000000000000000031"),
		"200", setBody(
			"5eed00000000000000000031",
			"5eed00000000000000000021",
			joinedTopic,
			"0",
			"["+questionQ1+","+questionQ2+"]",
		))
	assertCatalogBody(t,
		catalogGet(t, wiring,
			"/questions/5eed00000000000000000031?populateTopic=false"),
		"200", setBody(
			"5eed00000000000000000031",
			"5eed00000000000000000021",
			"5eed00000000000000000021", "0",
			"["+questionQ1+","+questionQ2+"]",
		))

	// A dangling topic reference preserves its id and has no joined topic.
	assertCatalogBody(t,
		catalogGet(t, wiring, "/questions/5eed00000000000000000034"),
		"200", setBody(
			"5eed00000000000000000034",
			"5eed00000000000000000029",
			"null",
			"0",
			"[]",
		))

	// GET /questions/topic/{slug} is unpopulated unless explicitly requested.
	written := `{"id":"seed-l1-q1","type":"written",` +
		`"prompt":"Explain two's complement.",` +
		`"targetConcepts":["twos-complement"],` +
		`"feedback":{"correct":"Correct feedback.",` +
		`"incorrect":"Incorrect feedback."},` +
		`"rubrics":{"keyPoints":["Invert bits.","Add one."],` +
		`"misconceptions":["Sign bit only."]}}`
	created := `{"id":"created-q1","type":"mcq",` +
		`"prompt":"What does CPU stand for?","options":[` +
		`{"id":"o1","text":"Central Processing Unit"},` +
		`{"id":"o2","text":"Computer Personal Unit"},` +
		`{"id":"o3","text":"Central Program Utility"}],` +
		`"correctOptionId":"o1","targetConcepts":["hardware-basics"],` +
		`"feedback":{"correct":"Correct.","incorrect":"Incorrect."},` +
		`"rubrics":{"keyPoints":["Central processing."],` +
		`"misconceptions":["Personal computer."]}}`
	topicSetList := "[" +
		setBody(
			"5eed00000000000000000031",
			"5eed00000000000000000021",
			"5eed00000000000000000021", "0", "["+questionQ1+","+questionQ2+"]",
		) + "," +
		setBody("5eed00000000000000000032",
			"5eed00000000000000000021", "5eed00000000000000000021", "1",
			"["+written+"]",
		) + "," +
		setBody("5eed00000000000000000033",
			"5eed00000000000000000021", "5eed00000000000000000021", "4",
			"["+created+"]",
		) + "]"

	assertCatalogBody(t,
		catalogGet(t, wiring, "/questions/topic/binary-basics"),
		"200", topicSetList)
	populatedList := "[" + setBody(
		"5eed00000000000000000031", "5eed00000000000000000021",
		joinedTopic, "0", "["+questionQ1+","+questionQ2+"]",
	) + "," + setBody("5eed00000000000000000032",
		"5eed00000000000000000021", joinedTopic, "1", "["+written+"]",
	) + "," + setBody("5eed00000000000000000033",
		"5eed00000000000000000021", joinedTopic, "4", "["+created+"]",
	) + "]"
	assertCatalogBody(t,
		catalogGet(t, wiring,
			"/questions/topic/binary-basics?populateTopic=true"),
		"200", populatedList)

	// Unknown topic slug is the pinned 404.
	assertCatalogBody(t,
		catalogGet(t, wiring, "/questions/topic/no-such-topic"),
		"404", `{"message":"Topic not found","error":"Not Found",`+
			`"statusCode":404}`)
}
