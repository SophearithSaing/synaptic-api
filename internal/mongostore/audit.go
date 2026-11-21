package mongostore

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/SophearithSaing/synaptic-api/internal/audit"
)

// AuditStore implements audit.Repository with aiLogs and liveQuestions.
type AuditStore struct {
	logs          *mongo.Collection
	liveQuestions *mongo.Collection
}

// NewAuditStore builds an AuditStore over the application database.
func NewAuditStore(database *mongo.Database) *AuditStore {
	return &AuditStore{logs: database.Collection("aiLogs"),
		liveQuestions: database.Collection("liveQuestions")}
}

// Create inserts one compatible aiLog and returns its ID.
func (store *AuditStore) Create(ctx context.Context, record audit.Record) (string, error) {
	now := time.Now().UTC()
	document := AILogDocument{Operation: string(record.Operation), AIModel: record.Model,
		Prompt: record.Prompt, Output: record.Output, CreatedAt: now, UpdatedAt: now}
	result, err := store.logs.InsertOne(ctx, document)
	if err != nil {
		return "", err
	}
	id, ok := result.InsertedID.(bson.ObjectID)
	if !ok {
		return "", fmt.Errorf("unexpected aiLog ID type %T", result.InsertedID)
	}
	return id.Hex(), nil
}

// LinkLiveQuestion links an aiLog to a live question by hex IDs.
func (store *AuditStore) LinkLiveQuestion(ctx context.Context, auditID string, liveQuestionID string) error {
	logID, err := bson.ObjectIDFromHex(auditID)
	if err != nil {
		return err
	}
	questionID, err := bson.ObjectIDFromHex(liveQuestionID)
	if err != nil {
		return err
	}
	_, err = store.logs.UpdateOne(ctx, bson.M{"_id": logID}, bson.M{"$set": bson.M{"liveQuestion": questionID}})
	return err
}

// List returns a deterministic, populated aiLog page.
func (store *AuditStore) List(ctx context.Context, page int64, limit int64) (audit.Page, error) {
	skip, err := auditSkip(page, limit)
	if err != nil {
		return audit.Page{}, err
	}
	cursor, err := store.logs.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{
		{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1},
	}).SetSkip(skip).SetLimit(limit))
	if err != nil {
		return audit.Page{}, err
	}
	defer cursor.Close(ctx)
	documents := make([]AILogDocument, 0)
	for cursor.Next(ctx) {
		var document AILogDocument
		if err := cursor.Decode(&document); err != nil {
			return audit.Page{}, err
		}
		documents = append(documents, document)
	}
	if err := cursor.Err(); err != nil {
		return audit.Page{}, err
	}
	total, err := store.logs.CountDocuments(ctx, bson.M{})
	if err != nil {
		return audit.Page{}, err
	}
	questions, err := store.findLiveQuestions(ctx, documents)
	if err != nil {
		return audit.Page{}, err
	}
	items := make([]audit.Record, len(documents))
	for index, document := range documents {
		items[index] = auditRecord(document, questions)
	}
	return audit.Page{Items: items, Total: total, Page: page, Limit: limit}, nil
}

// auditSkip bounds pagination arithmetic before MongoDB receives it.
func auditSkip(page int64, limit int64) (int64, error) {
	if page < 1 || limit < 1 || page-1 > math.MaxInt64/limit {
		return 0, fmt.Errorf("invalid audit pagination")
	}
	return (page - 1) * limit, nil
}

// findLiveQuestions batches live-question references and tolerates dangling IDs.
func (store *AuditStore) findLiveQuestions(ctx context.Context, logs []AILogDocument) (map[bson.ObjectID]audit.LiveQuestion, error) {
	ids := make([]bson.ObjectID, 0, len(logs))
	for _, log := range logs {
		if log.LiveQuestion != nil {
			ids = append(ids, *log.LiveQuestion)
		}
	}
	if len(ids) == 0 {
		return map[bson.ObjectID]audit.LiveQuestion{}, nil
	}
	cursor, err := store.liveQuestions.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	questions := make(map[bson.ObjectID]audit.LiveQuestion)
	for cursor.Next(ctx) {
		var document LiveQuestionDocument
		if err := cursor.Decode(&document); err != nil {
			return nil, err
		}
		questions[document.ID] = audit.LiveQuestion{ID: document.ID.Hex(), Question: document.Question,
			Level: document.Level, QuestionNumber: document.QuestionNumber, Status: document.Status}
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return questions, nil
}

// auditRecord converts legacy BSON into a response-compatible audit record.
func auditRecord(document AILogDocument, questions map[bson.ObjectID]audit.LiveQuestion) audit.Record {
	record := audit.Record{ID: document.ID.Hex(), Operation: audit.Operation(document.Operation),
		Model: document.AIModel, Prompt: document.Prompt, Output: document.Output,
		CreatedAt: document.CreatedAt}
	if document.LiveQuestion != nil {
		if question, exists := questions[*document.LiveQuestion]; exists {
			record.LiveQuestion = &question
		}
	}
	return record
}
