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
