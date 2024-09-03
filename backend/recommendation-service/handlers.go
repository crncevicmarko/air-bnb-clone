package main

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type KeyProduct struct{}

type RecommendHandler struct {
	logger *log.Logger
	repo   *RecommendRepo
}

func NewRecommendHandler(l *log.Logger, r *RecommendRepo) *RecommendHandler {
	return &RecommendHandler{l, r}
}

func (rh *RecommendHandler) Insert(res http.ResponseWriter, req *http.Request) {

	recommendation := req.Context().Value(KeyProduct{}).(*Recommend)
	err := rh.repo.WriteUser(recommendation)
	if err != nil {
		rh.logger.Println("error:", err.Error())
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	res.WriteHeader(http.StatusCreated)
}

func (rh *RecommendHandler) GetAllRecommendations(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	username := vars["username"]

	accoIds, err := rh.repo.GetAllRecommendations(username)
	if err != nil {
		rh.logger.Print("Database exception: ", err)
	}

	e := json.NewEncoder(res)
	err = e.Encode(accoIds)
	if err != nil {
		http.Error(res, "Unable to convert to json", http.StatusInternalServerError)
		rh.logger.Fatal("Unable to convert to json :", err)
		return
	}

}

func (rh *RecommendHandler) MiddlewareRecommendDeserialization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		recommend := &Recommend{}
		err := recommend.FromJSON(h.Body)
		if err != nil {
			http.Error(rw, "Unable to decode json", http.StatusBadRequest)
			rh.logger.Fatal(err)
			return
		}

		ctx := context.WithValue(h.Context(), KeyProduct{}, recommend)
		h = h.WithContext(ctx)

		next.ServeHTTP(rw, h)
	})
}
