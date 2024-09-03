package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gocql/gocql"
)

type ReservationByAccommodation struct {
	AccoId               string     `json:"accoId"`
	ReservationId        gocql.UUID `json:"reservationId"`
	HostId               string     `json:"userId"`
	NumberPeople         int        `json:"numberPeople"`
	PriceByPeople        int        `json:"priceByPeople"`
	PriceByAccommodation int        `json:"priceByAccommodation"`
	StartDate            time.Time  `json:"startDate"`
	EndDate              time.Time  `json:"endDate"`
}

type ReservationByUser struct {
	UserId         string    `json:"userId" validate:"required"`
	ReservationId  string    `json:"reservationId" validate:"required"`
	AccoId         string    `json:"accoId" validate:"required"`
	Price          int       `json:"price" validate:"required"`
	StartDate      time.Time `json:"startDate" validate:"required"`
	NumberOfPeople int       `json:"numberOfPeople"`
	EndDate        time.Time `json:"endDate" validate:"required"`
	IsDeleted      bool      `json:"isDeleted"`
}

type UserReservations struct {
	ReservationId  string    `json:"reservationId" validate:"required"`
	AccoId         string    `json:"accoId" validate:"required"`
	Price          int       `json:"price" validate:"required"`
	StartDate      time.Time `json:"startDate" validate:"required"`
	NumberOfPeople int       `json:"numberOfPeople"`
	EndDate        time.Time `json:"endDate" validate:"required"`
}

type ReservationDateByDate struct {
	AccoId                string    `json:"acco_id"`
	BeginAccomodationDate time.Time `json:"begin_accomodation_date"`
	EndAccomodationDate   time.Time `json:"end_accomodation_date"`
}

type ReservationDateByDateGet struct {
	AccoId string `json:"acco_id"`
}
type ReservationDate struct {
	BeginAccomodationDate time.Time `json:"begin_accomodation_date"`
	EndAccomodationDate   time.Time `json:"end_accomodation_date"`
}

type ReqToken struct {
	Token string `json:"token"`
}

type RequestId struct {
	UserId string `json:"userId"`
}

type ReservationsByAccommodation []*ReservationByAccommodation
type ReservationsByUser []*UserReservations
type ReservationDatesByAccomodationId []*ReservationDate
type ReservationDatesByDate []*ReservationDateByDate
type ReservationDatesByDateGet []*ReservationDateByDateGet

func ValidateReservationDateByDate(reservation *ReservationDateByDate) error {
	if reservation.AccoId == "" {
		return errors.New("accoId is required")
	}
	//if reservation.BeginAccomodationDate == nil {
	//	return errors.New("BeginAccomodationDate is required")
	//}
	//if reservation.EndAccomodationDate != nil {
	//	return errors.New("EndAccomodationDate is required")
	//}
	return nil
}

func ValidateReservationByAccommodation(reservation *ReservationByAccommodation) error {
	if reservation.HostId == "" {
		return errors.New("hostId is required")
	}
	if reservation.AccoId == "" {
		return errors.New("accoId is required")
	}
	if reservation.NumberPeople == 0 {
		return errors.New("number of people is required")
	}
	//if reservation.StartDate == nil {
	//	return errors.New("startDate is required")
	//}
	//if reservation.EndDate != nil {
	//	return errors.New("endDate is required")
	//}
	return nil
}

type ErrResp struct {
	URL        string
	Method     string
	StatusCode int
}

func (e ErrResp) Error() string {
	return fmt.Sprintf("error [status code %d] for request: HTTP %s\t%s", e.StatusCode, e.Method, e.URL)
}

func (o *ReservationByAccommodation) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(o)
}
func (o *ReservationDatesByAccomodationId) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(o)
}
func (o *ReservationsByAccommodation) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(o)
}
func (o *ReservationsByUser) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(o)
}
func (o *ReservationByUser) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(o)
}
func (o *ReservationDatesByDate) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(o)
}
func (o *ReservationDatesByDateGet) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(o)
}

func (o *ReservationByAccommodation) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(o)
}
func (o *ReservationByUser) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(o)
}
func (o *ReservationDatesByDate) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(o)
}
func (o *ReservationDatesByDateGet) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(o)
}
