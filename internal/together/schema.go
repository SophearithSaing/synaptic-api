package together

// JSONSchemaResponseFormat is Together's JSON schema response_format payload.
type JSONSchemaResponseFormat struct {
	Type       string     `json:"type"`
	JSONSchema JSONSchema `json:"json_schema"`
}

// JSONSchema is a named strict JSON schema.
type JSONSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Strict      bool           `json:"strict"`
	Schema      map[string]any `json:"schema"`
}

// GeneratedQuestionResponseFormat constrains question-generation completions.
var GeneratedQuestionResponseFormat = JSONSchemaResponseFormat{
	Type: "json_schema",
	JSONSchema: JSONSchema{
		Name:        "generated_live_question",
		Description: "A generated live-session question.",
		Strict:      true,
		Schema:      generatedQuestionSchema(),
	},
}

// WrittenEvaluationResponseFormat constrains written-grading completions.
var WrittenEvaluationResponseFormat = JSONSchemaResponseFormat{
	Type: "json_schema",
	JSONSchema: JSONSchema{
		Name:        "written_answer_evaluations",
		Description: "Evaluation results for submitted written answers.",
		Strict:      true,
		Schema:      writtenEvaluationSchema(),
	},
}

func generatedQuestionSchema() map[string]any {
	stringField := map[string]any{"type": "string"}
	stringArray := map[string]any{
		"type": "array", "items": map[string]any{"type": "string"},
	}
	option := map[string]any{
		"type": "object", "additionalProperties": false,
		"properties": map[string]any{
			"id": stringField, "text": stringField,
		},
		"required": []string{"id", "text"},
	}
	question := map[string]any{
		"type": "object", "additionalProperties": false,
		"properties": map[string]any{
			"id":              stringField,
			"type":            map[string]any{"type": "string", "enum": []string{"mcq", "written"}},
			"prompt":          stringField,
			"options":         map[string]any{"type": "array", "items": option},
			"correctOptionId": stringField,
			"targetConcepts":  stringArray,
			"feedback": map[string]any{
				"type": "object", "additionalProperties": false,
				"properties": map[string]any{
					"correct": stringField, "incorrect": stringField,
				},
				"required": []string{"correct", "incorrect"},
			},
			"rubrics": map[string]any{
				"type": "object", "additionalProperties": false,
				"properties": map[string]any{
					"keyPoints": stringArray, "misconceptions": stringArray,
				},
				"required": []string{"keyPoints", "misconceptions"},
			},
		},
		"required": []string{
			"id", "type", "prompt", "targetConcepts", "feedback", "rubrics",
		},
	}
	return map[string]any{
		"type": "object", "additionalProperties": false,
		"properties": map[string]any{"question": question},
		"required":   []string{"question"},
	}
}

func writtenEvaluationSchema() map[string]any {
	stringField := map[string]any{"type": "string"}
	stringArray := map[string]any{
		"type": "array", "items": map[string]any{"type": "string"},
	}
	evaluation := map[string]any{
		"type": "object", "additionalProperties": false,
		"properties": map[string]any{
			"questionId":    stringField,
			"score":         map[string]any{"type": "number", "minimum": 0, "maximum": 1},
			"correctAnswer": stringField,
			"feedback":      stringField,
			"strengths":     stringArray,
			"weaknesses":    stringArray,
		},
		"required": []string{
			"questionId", "score", "correctAnswer", "feedback", "strengths",
			"weaknesses",
		},
	}
	return map[string]any{
		"type": "object", "additionalProperties": false,
		"properties": map[string]any{
			"evaluations": map[string]any{"type": "array", "items": evaluation},
		},
		"required": []string{"evaluations"},
	}
}
