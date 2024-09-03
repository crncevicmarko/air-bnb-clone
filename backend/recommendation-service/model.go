package main

import (
	"encoding/json"
	"io"
)

type Recommend struct {
	Username string `json:"username"`
	ID       string `json:"accomoId"`
}

type User struct {
	Username string `json:"username"`
}

type Accommodation struct {
	Id string `json:"_id"`
}

func (rr *Recommend) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(rr)
}

func (rr *Recommend) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(rr)
}
