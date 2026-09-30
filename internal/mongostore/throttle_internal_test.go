package mongostore

import (
	"testing"
	"time"

	"github.com/SophearithSaing/synaptic-api/internal/web"
)

// TestThrottleDocumentStateMapping pins the stored-state to domain
// mapping in both block shapes.
func TestThrottleDocumentStateMapping(t *testing.T) {
	epoch := time.Unix(100, 0)
	expire := time.Unix(160, 0)
	blocked := time.Unix(300, 0)

	unblocked := throttleDocument{
		ID: "1.2.3.4|", Epoch: epoch, Expire: expire, Count: 7,
		PurgeAt: expire, Version: 9,
	}
	if state := unblocked.state(); state.Epoch != epoch ||
		state.Expire != expire || state.Count != 7 ||
		!state.BlockedUntil.IsZero() {
		t.Fatalf("unblocked mapping %+v", state)
	}

	blockedDocument := unblocked
	blockedDocument.BlockedUntil = &blocked
	blockedDocument.PurgeAt = blocked
	state := blockedDocument.state()
	if !state.BlockedUntil.Equal(blocked) {
		t.Fatalf("blocked mapping %+v", state)
	}
}

// TestNewThrottleDocumentPinsPurgeAndVersion pins insertion shape: the
// purge marker holds the later of horizon and block, blockedUntil is
// omitted without a block, and inserts start at version 1.
func TestNewThrottleDocumentPinsPurgeAndVersion(t *testing.T) {
	epoch := time.Unix(100, 0)
	expire := time.Unix(160, 0)

	unblocked := newThrottleDocument("1.2.3.4|", web.ThrottleState{
		Epoch: epoch, Expire: expire, Count: 1,
	})
	if unblocked.PurgeAt != expire || unblocked.Version != 1 ||
		unblocked.BlockedUntil != nil || unblocked.ID != "1.2.3.4|" {
		t.Fatalf("unblocked insert %+v", unblocked)
	}

	blocked := newThrottleDocument("1.2.3.4|", web.ThrottleState{
		Epoch: epoch, Expire: expire, Count: 4,
		BlockedUntil: epoch.Add(5 * time.Minute),
	})
	if !blocked.PurgeAt.Equal(epoch.Add(5 * time.Minute)) {
		t.Fatalf("blocked insert purge marker %v", blocked.PurgeAt)
	}
	if blocked.BlockedUntil == nil ||
		!blocked.BlockedUntil.Equal(epoch.Add(5*time.Minute)) {
		t.Fatalf("blocked insert blockedUntil %+v", blocked.BlockedUntil)
	}
}

// TestThrottleSetPinsCASPayload pins the compare-and-set payload: the
// next version always advances and a cleared block resets the field.
func TestThrottleSetPinsCASPayload(t *testing.T) {
	epoch := time.Unix(100, 0)
	expire := time.Unix(160, 0)

	payload := throttleSet(web.ThrottleState{
		Epoch: epoch, Expire: expire, Count: 5,
		BlockedUntil: epoch.Add(2 * time.Minute),
	}, 8)
	if payload["version"] != int64(8) || payload["count"] != 5 {
		t.Fatalf("payload %+v", payload)
	}
	if !payload["purgeAt"].(time.Time).Equal(epoch.Add(2 * time.Minute)) {
		t.Fatalf("payload purgeAt %+v", payload["purgeAt"])
	}

	cleared := throttleSet(web.ThrottleState{
		Epoch: epoch, Expire: expire, Count: 1,
	}, 2)
	if cleared["blockedUntil"] != nil {
		t.Fatalf("cleared block must null out, got %+v", cleared)
	}
}

// TestThrottleStateEqual pins the short-circuit comparison for
// unchanged states.
func TestThrottleStateEqual(t *testing.T) {
	epoch := time.Unix(100, 0)
	state := web.ThrottleState{
		Epoch: epoch, Expire: epoch.Add(time.Minute), Count: 3,
	}

	if !throttleStateEqual(state, state) {
		t.Fatal("a state must equal itself")
	}

	changed := state
	changed.Count++
	if throttleStateEqual(state, changed) {
		t.Fatal("an incremented count cannot equal the stored state")
	}
}
