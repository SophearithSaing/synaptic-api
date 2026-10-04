package mongostore_test

// joinedTopic is the question-set topic for binary-basics.
const joinedTopic = `{"id":"5eed00000000000000000021",` +
	`"title":"Binary Basics","slug":"binary-basics",` +
	`"description":"Binary numbers and arithmetic.","icon":"binary",` +
	`"tags":["binary","arithmetic"],` +
	`"category":` + categoryCore + `}`

// questionQ1 and questionQ2 are the pinned stored question bodies.
const questionQ1 = `{"id":"seed-l0-q1","type":"mcq",` +
	`"prompt":"What is 1 + 1 in binary?","options":[` +
	`{"id":"o1","text":"seed-l0-q1 option one"},` +
	`{"id":"o2","text":"seed-l0-q1 option two"},` +
	`{"id":"o3","text":"seed-l0-q1 option three"}],` +
	`"correctOptionId":"o1","targetConcepts":["binary-addition"],` +
	`"feedback":{"correct":"Correct feedback.",` +
	`"incorrect":"Incorrect feedback."},` +
	`"rubrics":{"keyPoints":["Key point."],` +
	`"misconceptions":["Misconception."]}}`

const questionQ2 = `{"id":"seed-l0-q2","type":"mcq",` +
	`"prompt":"What is 10 + 1 in binary?","options":[` +
	`{"id":"o1","text":"seed-l0-q2 option one"},` +
	`{"id":"o2","text":"seed-l0-q2 option two"},` +
	`{"id":"o3","text":"seed-l0-q2 option three"}],` +
	`"correctOptionId":"o2","targetConcepts":["binary-addition"],` +
	`"feedback":{"correct":"Correct feedback.",` +
	`"incorrect":"Incorrect feedback."},` +
	`"rubrics":{"keyPoints":["Key point."],` +
	`"misconceptions":["Misconception."]}}`

// categoryCoreAndNetwork is the sorted category list body.
const categoryCoreAndNetwork = `[` +
	`{"id":"5eed00000000000000000011","title":"Core Concepts",` +
	`"slug":"core-concepts","description":` +
	`"Foundational computing theory.","icon":"cpu"},` +
	`{"id":"5eed00000000000000000012","title":"Networking",` +
	`"slug":"networking","description":"Networks and protocols.",` +
	`"icon":"wifi"}]`

// categoryCore is one pinned category object.
const categoryCore = `{"id":"5eed00000000000000000011",` +
	`"title":"Core Concepts","slug":"core-concepts",` +
	`"description":"Foundational computing theory.","icon":"cpu"}`

// categoryNetwork is the second pinned category object.
const categoryNetwork = `{"id":"5eed00000000000000000012",` +
	`"title":"Networking","slug":"networking",` +
	`"description":"Networks and protocols.","icon":"wifi"}`

// topicNested wraps one topic DTO with a joined category object.
func topicNested(
	id, title, slug, description, icon string, tags string,
) string {
	return `{"id":"` + id + `","title":"` + title +
		`","slug":"` + slug + `","description":"` + description +
		`","icon":"` + icon + `","tags":` + tags +
		`,"category":` + categoryCore + `}`
}

// setBody builds one question-set body with a given topic value.
func setBody(id, topicID, topic, level string, questions string) string {
	return `{"id":"` + id + `","topicId":"` + topicID +
		`","topic":` + topic +
		`,"setType":"regular","level":` + level +
		`,"questions":` + questions +
		`,"createdAt":"2026-01-01T00:00:00.000Z",` +
		`"updatedAt":"2026-01-01T00:00:00.000Z"}`
}
