package token

import (
	"errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// Different types of error returned by the VerifyToken function
var (
	ErrInvalidToken = errors.New("token is invalid")
	ErrExpiredToken = errors.New("token has expired")
)

// Payload contains the payload data of the token
type Payload struct {
	ID        primitive.ObjectID `json:"id"`
	Username  string             `json:"username"`
	IssuedAt  time.Time          `json:"issued_at"`
	Role      string             `json:"role"`
	ExpiredAt time.Time          `json:"expired_at"`
}

// Needs to be in token folder
// NewPayload creates a new token payload with a specific username and duration
func NewPayload(id primitive.ObjectID, username string, role string, duration time.Duration) (*Payload, error) {
	//tokenID, err := uuid.NewRandom()
	//if err != nil {
	//	return nil, err
	//}

	payload := &Payload{
		ID:        id,
		Username:  username,
		Role:      role,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(duration),
	}
	return payload, nil
}

// Valid checks if the token payload is valid or not
func (payload *Payload) Valid() error {
	if time.Now().After(payload.ExpiredAt) {
		return ErrExpiredToken
	}
	return nil
}
