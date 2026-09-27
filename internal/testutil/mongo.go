// Package testutil provides shared helpers for integration tests.
package testutil

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// mongoImage is the MongoDB version used by integration tests. Version 8.2
// is the minimum that supports Linux kernels 6.19+ (SERVER-121912).
const mongoImage = "mongo:8.2"

// replicaSetTimeout bounds replica set initiation and primary election.
const replicaSetTimeout = 60 * time.Second

// StartMongo starts a single-node replica set MongoDB container and returns
// its connection URI. The test is skipped when running with -short. The
// replica set is required because transactions are under test.
func StartMongo(t *testing.T, ctx context.Context) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	container, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image: mongoImage,
				Cmd: []string{
					"mongod", "--replSet", "rs0", "--bind_ip_all",
				},
				ExposedPorts: []string{"27017/tcp"},
				WaitingFor: wait.ForLog("Waiting for connections").
					WithStartupTimeout(replicaSetTimeout),
			},
			Started: true,
		},
	)
	if err != nil {
		t.Fatalf("start mongo container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminate mongo container: %v", err)
		}
	})

	initiateReplicaSet(t, ctx, container)

	endpoint, err := container.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("resolve mongo endpoint: %v", err)
	}

	return "mongodb://" + endpoint + "/?directConnection=true"
}

// initiateReplicaSet runs rs.initiate and waits for a writable primary.
func initiateReplicaSet(
	t *testing.T,
	ctx context.Context,
	container testcontainers.Container,
) {
	t.Helper()
	deadline := time.Now().Add(replicaSetTimeout)

	initiate := `rs.initiate({_id: 'rs0', ` +
		`members: [{_id: 0, host: 'localhost:27017'}]})`
	for {
		code, _, err := container.Exec(
			ctx, []string{"mongosh", "--quiet", "--eval", initiate},
		)
		if err == nil && code == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("initiate replica set: %v (exit %d)", err, code)
		}
		time.Sleep(time.Second)
	}

	for {
		code, output, err := container.Exec(ctx, []string{
			"mongosh", "--quiet", "--eval", "db.hello().isWritablePrimary",
		})
		if err == nil && code == 0 {
			data, _ := io.ReadAll(output)
			if strings.Contains(string(data), "true") {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("replica set did not elect a primary")
		}
		time.Sleep(time.Second)
	}
}
