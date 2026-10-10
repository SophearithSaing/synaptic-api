package catalog_test

import (
	"net/http"
	"testing"
)

const handlerPopulatedTopic = `{"_id":"665f1e2b9d1a2c3b4d5e0004",` +
	`"title":"Binary Basics","slug":"binary-basics",` +
	`"description":"Binary numbers and arithmetic.","icon":"binary",` +
	`"tags":["binary","arithmetic"],` +
	`"category":"665f1e2b9d1a2c3b4d5e0003",` +
	`"createdAt":"2026-01-01T00:00:00Z",` +
	`"updatedAt":"2026-01-01T00:00:00Z","__v":3}`

// TestGetCategoriesFixtureShape pins the list response bytes sorted
// by title.
func TestGetCategoriesFixtureShape(t *testing.T) {
	handler, token := seededCatalog()

	response := catalogGet(t, handler, token, "/categories/categories")
	assertBody(t, response, http.StatusOK,
		`[{"id":"665f1e2b9d1a2c3b4d5e0003","title":"Core Concepts",`+
			`"slug":"core-concepts","description":`+
			`"Foundational computing theory.","icon":"cpu"},`+
			`{"id":"665f1e2b9d1a2c3b4d5e0009","title":"Networking",`+
			`"slug":"networking","description":"Networks and protocols.",`+
			`"icon":"wifi"}]`)
}

// TestGetCategoriesUnauth401 pins the failure body.
func TestGetCategoriesUnauth401(t *testing.T) {
	handler, _ := seededCatalog()

	response := catalogGet(t, handler, "", "/categories/categories")
	assertBody(t, response, http.StatusUnauthorized,
		`{"message":"Unauthorized","statusCode":401}`)
}

// TestGetTopicsFixtureShape pins the nested topic shape.
func TestGetTopicsFixtureShape(t *testing.T) {
	handler, token := seededCatalog()

	response := catalogGet(t, handler, token, "/topics")
	assertBody(t, response, http.StatusOK,
		`[{"id":"665f1e2b9d1a2c3b4d5e0004","title":"Binary Basics",`+
			`"slug":"binary-basics","description":`+
			`"Binary numbers and arithmetic.","icon":"binary",`+
			`"tags":["binary","arithmetic"],`+
			`"category":{`+
			`"id":"665f1e2b9d1a2c3b4d5e0003","title":"Core Concepts",`+
			`"slug":"core-concepts","description":`+
			`"Foundational computing theory.","icon":"cpu"}}]`)
}

// TestGetTopicsUnauth401 pins the unauth failure body.
func TestGetTopicsUnauth401(t *testing.T) {
	handler, _ := seededCatalog()

	response := catalogGet(t, handler, "", "/topics")
	assertBody(t, response, http.StatusUnauthorized,
		`{"message":"Unauthorized","statusCode":401}`)
}

// TestGetCategoryByIdVariants pins success, invalid, and 404.
func TestGetCategoryByIdVariants(t *testing.T) {
	handler, token := seededCatalog()

	response := catalogGet(t, handler, token,
		"/categories/665f1e2b9d1a2c3b4d5e0003")
	assertBody(t, response, http.StatusOK,
		`{"id":"665f1e2b9d1a2c3b4d5e0003","title":"Core Concepts",`+
			`"slug":"core-concepts","description":`+
			`"Foundational computing theory.","icon":"cpu"}`)

	invalid := catalogGet(t, handler, token, "/categories/not-an-id")
	assertBody(t, invalid, http.StatusBadRequest,
		`{"message":"Invalid MongoDB ObjectId","error":"Bad Request",`+
			`"statusCode":400}`)

	missing := catalogGet(t, handler, token,
		"/categories/665f1e2b9d1a2c3b4d5e0008")
	assertBody(t, missing, http.StatusNotFound,
		`{"message":"Category not found","error":"Not Found",`+
			`"statusCode":404}`)
}

// TestGetTopicByIdVariants pins success, invalid, and 404.
func TestGetTopicByIdVariants(t *testing.T) {
	handler, token := seededCatalog()

	response := catalogGet(t, handler, token,
		"/topics/665f1e2b9d1a2c3b4d5e0004")
	assertBody(t, response, http.StatusOK,
		`{"id":"665f1e2b9d1a2c3b4d5e0004","title":"Binary Basics",`+
			`"slug":"binary-basics","description":`+
			`"Binary numbers and arithmetic.","icon":"binary",`+
			`"tags":["binary","arithmetic"],`+
			`"category":{`+
			`"id":"665f1e2b9d1a2c3b4d5e0003","title":"Core Concepts",`+
			`"slug":"core-concepts","description":`+
			`"Foundational computing theory.","icon":"cpu"}}`)

	invalid := catalogGet(t, handler, token, "/topics/not-an-id")
	assertBody(t, invalid, http.StatusBadRequest,
		`{"message":"Invalid MongoDB ObjectId","error":"Bad Request",`+
			`"statusCode":400}`)

	missing := catalogGet(t, handler, token,
		"/topics/665f1e2b9d1a2c3b4d5e0008")
	assertBody(t, missing, http.StatusNotFound,
		`{"message":"Topic not found","error":"Not Found",`+
			`"statusCode":404}`)
}

// TestGetQuestionSetVariants pins the relation shape and errors.
func TestGetQuestionSetVariants(t *testing.T) {
	handler, token := seededCatalog()

	response := catalogGet(t, handler, token,
		"/questions/665f1e2b9d1a2c3b4d5e0006")
	assertBody(t, response, http.StatusOK,
		`{"id":"665f1e2b9d1a2c3b4d5e0006",`+
			`"topic":`+
			handlerPopulatedTopic+`,`+
			`"setType":"regular","level":0,"questions":[`+
			`{"id":"seed-l0-q1","type":"mcq",`+
			`"prompt":"What is 1 + 1 in binary?","options":[`+
			`{"id":"o1","text":"seed-l0-q1 option one"},`+
			`{"id":"o2","text":"seed-l0-q1 option two"},`+
			`{"id":"o3","text":"seed-l0-q1 option three"}],`+
			`"correctOptionId":"o1","targetConcepts":["binary-addition"],`+
			`"feedback":{"correct":"Correct feedback.","incorrect":`+
			`"Incorrect feedback."},"rubrics":{"keyPoints":["Key point."],`+
			`"misconceptions":["Misconception."]}}],`+
			`"createdAt":"2026-01-01T00:00:00Z",`+
			`"updatedAt":"2026-01-01T00:00:00Z"}`)

	invalid := catalogGet(t, handler, token, "/questions/not-an-id")
	assertBody(t, invalid, http.StatusBadRequest,
		`{"message":"Invalid MongoDB ObjectId","error":"Bad Request",`+
			`"statusCode":400}`)

	missing := catalogGet(t, handler, token,
		"/questions/665f1e2b9d1a2c3b4d5e0008")
	assertBody(t, missing, http.StatusNotFound,
		`{"message":"Question set not found","error":"Not Found",`+
			`"statusCode":404}`)
}

// TestGetQuestionSetsByTopic pins the joined relation shape.
func TestGetQuestionSetsByTopic(t *testing.T) {
	handler, token := seededCatalog()

	response := catalogGet(t, handler, token,
		"/questions/topic/binary-basics?populateTopic=true")
	assertBody(t, response, http.StatusOK,
		`[{"id":"665f1e2b9d1a2c3b4d5e0006",`+
			`"topic":`+handlerPopulatedTopic+`,"setType":"regular",`+
			`"level":0,"questions":[{"id":"seed-l0-q1","type":"mcq",`+
			`"prompt":"What is 1 + 1 in binary?","options":[`+
			`{"id":"o1","text":"seed-l0-q1 option one"},`+
			`{"id":"o2","text":"seed-l0-q1 option two"},`+
			`{"id":"o3","text":"seed-l0-q1 option three"}],`+
			`"correctOptionId":"o1","targetConcepts":["binary-addition"],`+
			`"feedback":{"correct":"Correct feedback.","incorrect":`+
			`"Incorrect feedback."},"rubrics":{"keyPoints":["Key point."],`+
			`"misconceptions":["Misconception."]}}],`+
			`"createdAt":"2026-01-01T00:00:00Z",`+
			`"updatedAt":"2026-01-01T00:00:00Z"}]`)

	notFound := catalogGet(t, handler, token,
		"/questions/topic/no-such-topic")
	assertBody(t, notFound, http.StatusNotFound,
		`{"message":"Topic not found","error":"Not Found",`+
			`"statusCode":404}`)
}
