package mongostore

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// UserDocument is the exact users BSON representation, including the
// Mongoose additive fields.
type UserDocument struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Username  string        `bson:"username"`
	Email     string        `bson:"email"`
	Password  string        `bson:"password"`
	Role      string        `bson:"role"`
	CreatedAt time.Time     `bson:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"`
	Version   int           `bson:"__v"`
}

// AuthSessionDocument is the exact authSessions BSON representation:
// userId and expiresAt are required, revokedAt is optional, and
// refreshTokenHash is excluded from default queries.
type AuthSessionDocument struct {
	ID               bson.ObjectID `bson:"_id,omitempty"`
	UserID           bson.ObjectID `bson:"userId"`
	RefreshTokenHash string        `bson:"refreshTokenHash"`
	ExpiresAt        time.Time     `bson:"expiresAt"`
	RevokedAt        *time.Time    `bson:"revokedAt,omitempty"`
	CreatedAt        time.Time     `bson:"createdAt"`
	UpdatedAt        time.Time     `bson:"updatedAt"`
	Version          int           `bson:"__v"`
}

// Catalog read support: questions are loose embedded documents, so the
// raw types below keep field order and tolerate every legacy shape.

// CategoryDocument is the exact categories BSON representation.
type CategoryDocument struct {
	ID          bson.ObjectID `bson:"_id"`
	Title       string        `bson:"title"`
	Slug        string        `bson:"slug"`
	Description string        `bson:"description"`
	Icon        string        `bson:"icon"`
}

// TopicDocument is the exact topics BSON representation.
type TopicDocument struct {
	ID          bson.ObjectID `bson:"_id"`
	Title       string        `bson:"title"`
	Slug        string        `bson:"slug"`
	Description string        `bson:"description"`
	Icon        string        `bson:"icon"`
	Tags        []string      `bson:"tags"`
	// Category is typed loosely: current documents store an ObjectId,
	// legacy documents may store a hex string.
	Category  bson.RawValue `bson:"category,omitempty"`
	CreatedAt time.Time     `bson:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"`
	Version   int           `bson:"__v"`
}

// QuestionSetDocument is the exact questionSets BSON representation
// with loose embedded questions.
type QuestionSetDocument struct {
	ID bson.ObjectID `bson:"_id"`
	// Topic is typed loosely: ObjectId or, in some legacy documents, a
	// hex string.
	Topic   bson.RawValue `bson:"topic"`
	SetType string        `bson:"setType"`
	Level   int64         `bson:"level"`
	// Questions stores each embedded question loosely to preserve
	// field order and tolerate every stored legacy shape.
	Questions bson.RawValue `bson:"questions"`
	CreatedAt time.Time     `bson:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}
