package main

import (
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Role string

const (
	Host  Role = "HOST"
	Guest Role = "GUEST"
)

type User struct {
	ID        string   `bson:"_id,omitempty" json:"userId"`
	Username  string   `bson:"username,omitempty" json:"username"`
	Email     string   `bson:"email,omitempty" json:"email"`
	Role      Role     `bson:"role,omitempty" json:"role" `
	FirstName string   `bson:"firstname,omitempty" json:"firstname"`
	LastName  string   `bson:"lastname,omitempty" json:"lastname"`
	Location  Location `bson:"location,omitempty,inline" json:"location"`
}

type Location struct {
	Country      string `bson:"country,omitempty" json:"country"`
	City         string `bson:"city,omitempty" json:"city"`
	StreetName   string `bson:"streetName,omitempty" json:"streetName"`
	StreetNumber string `bson:"streetNumber,omitempty" json:"streetNumber"`
}

type ResponseUser struct {
	Username  string   `bson:"username,omitempty" json:"username"`
	Email     string   `bson:"email,omitempty" json:"email"`
	Role      Role     `bson:"role,omitempty" json:"role"`
	FirstName string   `bson:"firstname,omitempty" json:"firstname"`
	LastName  string   `bson:"lastname,omitempty" json:"lastname"`
	Location  Location `bson:"location,omitempty,inline" json:"location"`
}

type VerifyEmail struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Username   string             `bson:"username,omitempty" json:"username" validate:"required"`
	Email      string             `bson:"email,omitempty" json:"email" validate:"required"`
	SecretCode string             `bson:"secretCode,omitempty" json:"secretCode" validate:"required"`
	IsUsed     bool               `bson:"isUsed" json:"isUsed" validate:"required"`
	CreatedAt  time.Time          `bson:"createdAt,omitempty" json:"createdAt" validate:"required"`
	ExpiredAt  time.Time          `bson:"expiredAt,omitempty" json:"expiredAt" validate:"required"`
}

type HostGrade struct {
	ID        string `bson:"id,omitempty" json:"id"`
	UserId    string `bson:"userId,omitempty" json:"userId"`
	HostId    string `bson:"hostId,omitempty" json:"hostId"`
	CreatedAt string `bson:"createdAt,omitempty" json:"createdAt"`
	Grade     int    `bson:"grate,omitempty" json:"grade"`
	// IsDeleted    bool      `bson:"isUsed" json:"isUsed" validate:"required"`
}
type ReqToken struct {
	Token string `json:"token"`
}

type RequestId struct {
	UserId string `json:"userId"`
}

type Users []*User

type AverageGrade struct {
	UserId       string  `json:"userId"`
	AverageGrade float64 `json:"averageGrade"`
}

type ErrResp struct {
	URL        string
	Method     string
	StatusCode int
}

func (e ErrResp) Error() string {
	return fmt.Sprintf("error [status code %d] for request: HTTP %s\t%s", e.StatusCode, e.Method, e.URL)
}

func ValidateUser(user *User) error {
	if user.Username == "" {
		return errors.New("username is required")
	}
	if user.Email == "" {
		return errors.New("email is required")
	}
	if user.FirstName == "" {
		return errors.New("firsName is required")
	}
	if user.LastName == "" {
		return errors.New("lastName is required")
	}
	if user.Location.City == "" {
		return errors.New("city is required")
	}
	if user.Location.Country == "" {
		return errors.New("country is required")
	}
	if user.Location.StreetName == "" {
		return errors.New("streetName is required")
	}
	if user.Location.StreetNumber == "" {
		return errors.New("streetNumber is required")
	}

	return nil
}

func ValidateHostGrade(hostGrade *HostGrade) error {
	if hostGrade.UserId == "" {
		return errors.New("userId is required")
	}
	if hostGrade.HostId == "" {
		return errors.New("hostId is required")
	}
	if hostGrade.Grade == 0 {
		return errors.New("grade is required")
	}
	if hostGrade.Grade <= 0 || hostGrade.Grade > 5 {
		return errors.New("grade must be in the range of 1 to 5")
	}

	return nil
}
