package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

type Accommodation struct {
	ID           string    `bson:"_id,omitempty" json:"_id"`
	Name         string    `bson:"name,omitempty" json:"name"`
	Location     Location  `bson:"location,omitempty,inline" json:"location"`
	Amenities    []Amenity `bson:"amenities,omitempty" json:"amenities"`
	MinGuests    int       `bson:"minGuests,omitempty" json:"minGuests"`
	MaxGuests    int       `bson:"maxGuests,omitempty" json:"maxGuests"`
	Username     string    `bson:"username,omitempty" json:"username"`
	AverageGrade float64   `bson:"averageGrade,omitempty" json:"averageGrade"`
	Images       []string  `bson:"images,omitempty" json:"images"`
	Approved     string    `bson:"approved,omitempty"`
	// Availability bool   `json:"availability"`
	// Details      string `json:"details"`
	//Price     string 			`bson:"price,omitempty" json:"price"`
}
type Accommodation2 struct {
	ID                   string    `bson:"_id,omitempty" json:"_id"`
	Name                 string    `bson:"name,omitempty" json:"name"`
	Location             Location  `bson:"location,omitempty,inline" json:"location"`
	Amenities            []Amenity `bson:"amenities,omitempty" json:"amenities"`
	MinGuests            int       `bson:"minGuests,omitempty" json:"minGuests"`
	MaxGuests            int       `bson:"maxGuests,omitempty" json:"maxGuests"`
	Username             string    `bson:"username,omitempty" json:"username"`
	AverageGrade         float64   `bson:"averageGrade,omitempty" json:"averageGrade"`
	Images               []string  `bson:"images,omitempty" json:"images"`
	NumberPeople         int       `json:"numberPeople"`
	PriceByPeople        int       `json:"priceByPeople"`
	PriceByAccommodation int       `json:"priceByAccommodation"`
	StartDate            string    `json:"startDate"`
	EndDate              string    `json:"endDate"`
	// Availability bool   `json:"availability"`
	// Details      string `json:"details"`
	//Price     string 			`bson:"price,omitempty" json:"price"`
}

type ReservationByAccommodation struct { // ako -||-1 radi onda ovaj treba obrisati
	AccoId               string `json:"accoId"`
	NumberPeople         int    `json:"numberPeople"`
	PriceByPeople        int    `json:"priceByPeople"`
	PriceByAccommodation int    `json:"priceByAccommodation"`
	StartDate            string `json:"startDate"`
	EndDate              string `json:"endDate"`
}
type ReservationByAccommodation1 struct {
	AccoId               string    `json:"accoId"`
	HostId               string    `json:"hostId"`
	NumberPeople         int       `json:"numberPeople"`
	PriceByPeople        int       `json:"priceByPeople"`
	PriceByAccommodation int       `json:"priceByAccommodation"`
	StartDate            time.Time `json:"startDate"`
	EndDate              time.Time `json:"endDate"`
}

type ReservationByUser struct {
	AccoId         string    `json:"accoId" validate:"required"`
	Price          int       `json:"price" validate:"required"`
	StartDate      time.Time `json:"startDate" validate:"required"`
	NumberOfPeople int       `json:"numberOfPeople"`
	EndDate        time.Time `json:"endDate" validate:"required"`
}

type UserReservations struct {
	ReservationId  string    `json:"reservationId" validate:"required"`
	AccoId         string    `json:"accoId" validate:"required"`
	Price          int       `json:"price" validate:"required"`
	StartDate      time.Time `json:"startDate" validate:"required"`
	NumberOfPeople int       `json:"numberOfPeople"`
	EndDate        time.Time `json:"endDate" validate:"required"`
}

type Location struct {
	Country      string `bson:"country,omitempty" json:"country"`
	City         string `bson:"city,omitempty" json:"city"`
	StreetName   string `bson:"streetName,omitempty" json:"streetName"`
	StreetNumber string `bson:"streetNumber,omitempty" json:"streetNumber"`
}

type Amenity string

const (
	WIFI             Amenity = "WIFI"
	Heating          Amenity = "Heating"
	Air_conditioning Amenity = "Air conditioning"
	Kitchen          Amenity = "Kitchen"
	TV               Amenity = "TV"
	Washer           Amenity = "Washer"
)

type Accommodations []*Accommodation
type AccommodationGrades []*AccommodationGrade
type ReservationsByUser []*UserReservations

type ReqToken struct {
	Token string `json:"token"`
}

type AccommodationGrade struct {
	ID              string `bson:"_id,omitempty" json:"id"`
	UserId          string `bson:"userId,omitempty" json:"userId"`
	AccommodationId string `bson:"accommodationId,omitempty" json:"accommodationId"`
	CreatedAt       string `bson:"createdAt,omitempty" json:"createdAt"`
	Grade           int    `bson:"grade,omitempty" json:"grade"`
}

type ReqList struct {
	List []string `json:"list"`
}

type ErrResp struct {
	URL        string
	Method     string
	StatusCode int
}

func (e ErrResp) Error() string {
	return fmt.Sprintf("error [status code %d] for request: HTTP %s\t%s", e.StatusCode, e.Method, e.URL)
}

func (req *ReqList) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(req)
}

func (req *ReqList) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(req)
}

func (as *Accommodations) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(as)
}

func (a *Accommodation) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(a)
}

func (a *Accommodation) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(a)
}

func (a *Accommodation2) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(a)
}

func (a *AccommodationGrades) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(a)
}

func ValidateAccommodation(accommodation *Accommodation) error {
	if accommodation.MaxGuests == 0 {
		return errors.New("max guests is required")
	}
	if accommodation.MinGuests == 0 {
		return errors.New("min guests is required")
	}
	if accommodation.Name == "" {
		return errors.New("name is required")
	}
	if accommodation.Username == "" {
		return errors.New("username is required")
	}
	if accommodation.Location.StreetName == "" {
		return errors.New("street name is required")
	}
	if accommodation.Location.Country == "" {
		return errors.New("country is required")
	}
	if accommodation.Location.StreetNumber == "" {
		return errors.New("street number is required")
	}
	return nil
}

func ValidateAccommodationGrade(accommodationGrade *AccommodationGrade) error {
	// if accommodationGrade.UserId == "" {
	// 	return errors.New("userId is required")
	// }
	if accommodationGrade.AccommodationId == "" {
		return errors.New("hostId is required")
	}
	if accommodationGrade.Grade == 0 {
		return errors.New("grade is required")
	}
	if accommodationGrade.Grade <= 0 || accommodationGrade.Grade > 5 {
		return errors.New("grade must be in the range of 1 to 5")
	}

	return nil
}
