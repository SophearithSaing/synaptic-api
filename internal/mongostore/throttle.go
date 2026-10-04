package mongostore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/SophearithSaing/synaptic-api/internal/web"
)

// ThrottleStore implements web.ThrottleStore over the throttles
// collection. State updates of one key follow a compare-and-set loop
// on a stored version, so processes on every replica serialize their
// window counters against one authoritative document instead of an
// unsafe read-write pair.
type ThrottleStore struct {
	windows *mongo.Collection
}

// NewThrottleStore builds the shared throttle store over a database.
func NewThrottleStore(database *mongo.Database) *ThrottleStore {
	return &ThrottleStore{
		windows: database.Collection("throttles"),
	}
}

// throttleDocument is the stored fixed-window state of one rate limit
// key. Version serializes compare-and-set updates and purgeAt drives
// the TTL index, marking the later of the window horizon and an active
// block.
type throttleDocument struct {
	ID           string     `bson:"_id"`
	Epoch        time.Time  `bson:"epoch"`
	Expire       time.Time  `bson:"expire"`
	Count        int        `bson:"count"`
	BlockedUntil *time.Time `bson:"blockedUntil,omitempty"`
	PurgeAt      time.Time  `bson:"purgeAt"`
	Version      int64      `bson:"version"`
}

// throttleUpdateAttempts bounds one Update's compare-and-set retries
// before the caller decides the failure policy.
const throttleUpdateAttempts = 5

// errThrottleContention marks persistent compare-and-set conflicts on
// one hot key.
var errThrottleContention = errors.New("throttle update lost every race")

// Update implements web.ThrottleStore. Every attempt re-reads the
// current version, applies the fixed-window mutation, and persists it
// guarded by the observed version; a lost race retries from the fresh
// snapshot, so concurrent updates of one key can never overwrite a
// counted hit.
func (s *ThrottleStore) Update(
	ctx context.Context,
	key string,
	mutate func(state web.ThrottleState) web.ThrottleState,
) (web.ThrottleState, error) {
	var lastConflict error

	for range throttleUpdateAttempts {
		var document throttleDocument
		err := s.windows.FindOne(ctx, bson.M{"_id": key}).Decode(&document)
		if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			return web.ThrottleState{}, fmt.Errorf(
				"load throttle %s: %w", key, err,
			)
		}

		exists := err == nil
		var current web.ThrottleState
		var version int64
		if exists {
			current, version = document.state(), document.Version
		}

		next := mutate(current)
		if exists && throttleStateEqual(current, next) {
			// Rejected hits leave the stored state untouched.
			return next, nil
		}

		if !exists {
			insert := newThrottleDocument(key, next)
			if _, err := s.windows.InsertOne(ctx, insert); err != nil {
				if mongo.IsDuplicateKeyError(err) {
					lastConflict = err
					continue
				}

				return web.ThrottleState{}, fmt.Errorf(
					"insert throttle %s: %w", key, err,
				)
			}

			return next, nil
		}

		result, err := s.windows.UpdateOne(
			ctx,
			bson.M{"_id": key, "version": version},
			throttleSet(next, version+1),
		)
		if err != nil {
			return web.ThrottleState{}, fmt.Errorf(
				"update throttle %s: %w", key, err,
			)
		}
		if result.MatchedCount == 1 {
			return next, nil
		}

		lastConflict = errThrottleContention
	}

	return web.ThrottleState{}, fmt.Errorf(
		"update throttle %s: %w", key, lastConflict,
	)
}

// Size implements web.ThrottleStore.
func (s *ThrottleStore) Size(ctx context.Context) (int, error) {
	count, err := s.windows.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("count throttles: %w", err)
	}

	return int(count), nil
}

// Prune implements web.ThrottleStore: windows outside an open block
// and past their counting horizon as of now are reclaimed; the TTL
// index reclaims them independently in production.
func (s *ThrottleStore) Prune(
	ctx context.Context,
	now time.Time,
) (int64, error) {
	deleted, err := s.windows.DeleteMany(ctx, bson.M{
		"expire":       bson.M{"$lt": now},
		"blockedUntil": bson.M{"$not": bson.M{"$gt": now}},
	})
	if err != nil {
		return 0, fmt.Errorf("prune throttles: %w", err)
	}

	return deleted.DeletedCount, nil
}

// state converts the stored window into the domain state. A missing
// blockedUntil reads as no block.
func (document throttleDocument) state() web.ThrottleState {
	blockedUntil := time.Time{}
	if document.BlockedUntil != nil {
		blockedUntil = *document.BlockedUntil
	}

	return web.ThrottleState{
		Epoch:        document.Epoch,
		Expire:       document.Expire,
		Count:        document.Count,
		BlockedUntil: blockedUntil,
	}
}

// newThrottleDocument builds the first stored version of a window,
// setting the purge marker to the later of the horizon and the block.
func newThrottleDocument(
	key string,
	state web.ThrottleState,
) throttleDocument {
	document := throttleDocument{
		ID:      key,
		Epoch:   state.Epoch,
		Expire:  state.Expire,
		Count:   state.Count,
		PurgeAt: throttlePurgeAt(state),
		Version: 1,
	}
	if !state.BlockedUntil.IsZero() {
		blocked := state.BlockedUntil
		document.BlockedUntil = &blocked
	}

	return document
}

// throttleSet builds the compare-and-set payload for the next state
// and version.
func throttleSet(next web.ThrottleState, nextVersion int64) bson.M {
	set := bson.M{
		"epoch":   next.Epoch,
		"expire":  next.Expire,
		"count":   next.Count,
		"purgeAt": throttlePurgeAt(next),
		"version": nextVersion,
	}
	if next.BlockedUntil.IsZero() {
		set["blockedUntil"] = nil
	} else {
		set["blockedUntil"] = next.BlockedUntil
	}

	return set
}

// throttlePurgeAt marks when a window is eligible for the TTL index:
// the later of the counting horizon and an active block.
func throttlePurgeAt(state web.ThrottleState) time.Time {
	purgeAt := state.Expire
	if state.BlockedUntil.After(purgeAt) {
		purgeAt = state.BlockedUntil
	}

	return purgeAt
}

// throttleStateEqual reports whether two window states carry the same
// rules, treating zero instants as unordered.
func throttleStateEqual(a, b web.ThrottleState) bool {
	if a.Count != b.Count {
		return false
	}

	return a.Epoch.Equal(b.Epoch) && a.Expire.Equal(b.Expire) &&
		a.BlockedUntil.Equal(b.BlockedUntil)
}
