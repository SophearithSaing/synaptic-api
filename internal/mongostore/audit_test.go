package mongostore_test

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/SophearithSaing/synaptic-api/internal/audit"
	"github.com/SophearithSaing/synaptic-api/internal/catalog"
	"github.com/SophearithSaing/synaptic-api/internal/mongostore"
)

func TestAuditStorePersistencePaginationAndJoin(t *testing.T) {
	ctx := context.Background()
	client, err := mongostore.Connect(ctx, startMongo(t, ctx))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	database := client.Database("audittest")
	store := mongostore.NewAuditStore(database)
	questionID := bson.NewObjectID()
	_, err = database.Collection("liveQuestions").InsertOne(ctx, mongostore.LiveQuestionDocument{ID: questionID, Question: catalog.Question{ID: "q1", Type: "mcq"}, Level: 1, QuestionNumber: 2, Status: "pending"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	linkedID := bson.NewObjectID()
	danglingID := bson.NewObjectID()
	_, err = database.Collection("aiLogs").InsertMany(ctx, []any{
		mongostore.AILogDocument{ID: linkedID, Operation: "question-generation", AIModel: "m", Prompt: "p", Output: "o", LiveQuestion: &questionID, CreatedAt: now, UpdatedAt: now},
		mongostore.AILogDocument{ID: bson.NewObjectID(), Operation: "written-grading", AIModel: "m", Prompt: "p2", Output: "o2", LiveQuestion: &danglingID, CreatedAt: now.Add(time.Second), UpdatedAt: now},
	})
	if err != nil {
		t.Fatal(err)
	}
	page, err := store.List(ctx, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].Operation != "written-grading" || page.Items[0].LiveQuestion != nil {
		t.Fatalf("page=%#v", page)
	}
	page, err = store.List(ctx, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].LiveQuestion == nil || page.Items[0].LiveQuestion.ID != questionID.Hex() {
		t.Fatalf("joined page=%#v", page)
	}
	id, err := store.Create(ctx, audit.Record{Operation: audit.OperationWrittenGrading, Model: "m", Prompt: "p", Output: "o"})
	if err != nil || id == "" {
		t.Fatalf("create=%q err=%v", id, err)
	}
	if err := store.LinkLiveQuestion(ctx, id, questionID.Hex()); err != nil {
		t.Fatalf("LinkLiveQuestion() error = %v", err)
	}
}
