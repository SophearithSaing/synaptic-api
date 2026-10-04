package mongostore_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
	"github.com/SophearithSaing/synaptic-api/internal/identity"
	"github.com/SophearithSaing/synaptic-api/internal/mongostore"
	"github.com/SophearithSaing/synaptic-api/internal/testutil"
)

// seedLegacyCatalog inserts the tolerated legacy documents: a
// string-typed topic category, a dangling category reference, and a
// legacy question set.
func seedLegacyCatalog(
	t *testing.T,
	ctx context.Context,
	database *mongo.Database,
) {
	t.Helper()

	users := database.Collection("users")
	if _, err := users.InsertOne(ctx, map[string]any{
		"_id":      objectIDOf(t, "5eed00000000000000000001"),
		"username": "student",
		"email":    "student@example.com",
		"password": "supersafeseed",
		"role":     "user",
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	legacyCategoryID := objectIDOf(t, "5eed00000000000000000011")

	topics := database.Collection("topics")
	if _, err := topics.InsertOne(ctx, bson.D{
		{Key: "_id", Value: objectIDOf(t, "5eed00000000000000000025")},
		{Key: "title", Value: "Legacy Topic"},
		{Key: "slug", Value: "legacy-topic"},
		{Key: "description", Value: "Stored with a string reference."},
		{Key: "icon", Value: "archive"},
		{Key: "tags", Value: bson.A{"legacy"}},
		{Key: "category", Value: "5eed00000000000000000011"},
		{Key: "createdAt", Value: seedDate},
		{Key: "updatedAt", Value: seedDate},
		{Key: "__v", Value: 0},
	}); err != nil {
		t.Fatalf("seed legacy topic: %v", err)
	}

	questionSets := database.Collection("questionSets")
	if _, err := questionSets.InsertOne(ctx, bson.D{
		{Key: "_id", Value: objectIDOf(t, "5eed00000000000000000035")},
		{
			Key:   "topic",
			Value: objectIDOf(t, "5eed00000000000000000025"),
		},
		{Key: "setType", Value: "primary"},
		{Key: "level", Value: 0},
		{Key: "questions", Value: bson.A{
			bson.D{
				{Key: "id", Value: "legacy-q1"},
				{Key: "type", Value: "written"},
				{Key: "prompt", Value: "Legacy prompt."},
				{Key: "options", Value: nil},
				{Key: "correctOptionId", Value: nil},
				{Key: "targetConcepts", Value: bson.A{"legacy"}},
				{Key: "feedback", Value: bson.D{
					{Key: "correct", Value: "Correct."},
					{Key: "incorrect", Value: "Incorrect."},
					{Key: "sampleAnswer", Value: "Stored sample."},
					{
						Key:   "evaluationGuidance",
						Value: "Stored guidance.",
					},
				}},
				{Key: "rubrics", Value: bson.D{
					{Key: "keyPoints", Value: bson.A{"Point."}},
					{Key: "misconceptions", Value: bson.A{"Trap."}},
				}},
				{Key: "hints", Value: bson.A{"Stored hint."}},
			},
		}},
		{Key: "createdAt", Value: seedDate},
		{Key: "updatedAt", Value: seedDate},
		{Key: "__v", Value: 0},
	}); err != nil {
		t.Fatalf("seed legacy question set: %v", err)
	}
	_ = legacyCategoryID
}

// newLegacyWiring starts a container with legacy shapes seeded.
func newLegacyWiring(t *testing.T) *catalogWiring {
	t.Helper()

	ctx := context.Background()
	uri := testutil.StartMongo(t, ctx)

	client, err := mongostore.Connect(ctx, uri)
	if err != nil {
		t.Fatalf("connect mongo: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		_ = client.Disconnect(cleanupCtx)
	})

	database := client.Database("legacytest")
	if err := testutil.EnsureIdentityIndexes(ctx, database); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}
	seedLegacyCatalog(t, ctx, database)

	issuer := identity.NewTokenIssuer(
		"secret", "synaptic", "synaptic-client", time.Hour,
	)
	store := mongostore.NewIdentityStore(database)
	catalogStore := mongostore.NewCatalogStore(database)

	token, err := issuer.Issue(
		"5eed00000000000000000001", "student@example.com", "student", time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	catalog.NewHandler(catalogStore, identity.NewAuthenticator(
		issuer, store,
	)).Mount(mux)

	return &catalogWiring{mux: mux, token: token}
}

// TestCatalogLegacyTolerances pins unchanged legacy shapes on read.
func TestCatalogLegacyTolerances(t *testing.T) {
	wiring := newLegacyWiring(t)

	// A joined topic with a legacy string category leaves the category
	// item unresolved.
	assertCatalogBody(t,
		catalogGet(t, wiring, "/questions/5eed00000000000000000035"),
		"200", `{"id":"5eed00000000000000000035",`+
			`"topicId":"5eed00000000000000000025","topic":{`+
			`"id":"5eed00000000000000000025","title":"Legacy Topic",`+
			`"slug":"legacy-topic","description":`+
			`"Stored with a string reference.","icon":"archive",`+
			`"tags":["legacy"],"category":null},`+
			`"setType":"primary","level":0,"questions":[`+
			`{"id":"legacy-q1","type":"written","prompt":"Legacy prompt.",`+
			`"options":null,"correctOptionId":null,`+
			`"targetConcepts":["legacy"],"feedback":{`+
			`"correct":"Correct.","incorrect":"Incorrect.",`+
			`"sampleAnswer":"Stored sample.",`+
			`"evaluationGuidance":"Stored guidance."},`+
			`"rubrics":{"keyPoints":["Point."],`+
			`"misconceptions":["Trap."]},"hints":["Stored hint."]}],`+
			`"createdAt":"2026-01-01T00:00:00.000Z",`+
			`"updatedAt":"2026-01-01T00:00:00.000Z"}`)

	// The topic DTO tolerates the unresolved category with a null
	// nested object through the pinned shape.
	assertCatalogBody(t,
		catalogGet(t, wiring, "/topics/5eed00000000000000000025"),
		"200", `{"id":"5eed00000000000000000025","title":"Legacy Topic",`+
			`"slug":"legacy-topic","description":`+
			`"Stored with a string reference.","icon":"archive",`+
			`"tags":["legacy"],"category":null}`)
}
