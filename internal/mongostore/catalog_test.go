package mongostore_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
	"github.com/SophearithSaing/synaptic-api/internal/identity"
	"github.com/SophearithSaing/synaptic-api/internal/mongostore"
	"github.com/SophearithSaing/synaptic-api/internal/testutil"
)

// catalogWiring is the catalog routes over one seeded database.
type catalogWiring struct {
	mux   *http.ServeMux
	token string
}

// newCatalogWiring starts Mongo, seeds catalog data, and wires the
// routes with a real bearer identity.
func newCatalogWiring(t *testing.T) *catalogWiring {
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

	database := client.Database("catalogtest")
	if err := testutil.EnsureIdentityIndexes(ctx, database); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}
	users := database.Collection("users")
	password := "supersafeseed"
	for _, user := range []map[string]any{
		{
			"_id":      objectIDOf(t, "5eed00000000000000000001"),
			"username": "student",
			"email":    "student@example.com",
			"password": password,
			"role":     "user",
		},
		{
			"_id":      objectIDOf(t, "5eed00000000000000000002"),
			"username": "admin",
			"email":    "admin@example.com",
			"password": password,
			"role":     "admin",
		},
	} {
		if _, err := users.InsertOne(ctx, user); err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	seedFixturesCatalog(t, ctx, database)

	issuer := identity.NewTokenIssuer(
		"secret", "synaptic", "synaptic-client", time.Hour,
	)
	store := mongostore.NewIdentityStore(database)
	catalogStore := mongostore.NewCatalogStore(database)

	token, err := issuer.Issue(
		"5eed00000000000000000001",
		"student@example.com",
		"student",
		time.Now(),
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
