package token

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Needs to be in token folder
// Maker is an interface for managing tokens
type Maker interface {
	// CreateToken creates a new token for a specific username and duration
	CreateToken(id primitive.ObjectID, username string, role string, duration time.Duration) (string, *Payload, error)

	// VerifyToken checks if the token is valid or not
	VerifyToken(token string) (*Payload, error)
}
