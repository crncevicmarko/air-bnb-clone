package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	// "log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sony/gobreaker"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/thanhpk/randstr"
	"github.com/vukasinc25/fst-airbnb/handlers"
	"github.com/vukasinc25/fst-airbnb/token"
)

const (
	AccessTokenKey = "accessToken"
)

type KeyProduct struct{}

type AccoHandler struct {
	logger         *log.Logger
	db             *AccoRepo
	storageHandler *handlers.StorageHandler
	tracer         trace.Tracer
	orchestrator   *CreateAccommodationOrchestrator
}

func NewAccoHandler(l *log.Logger, r *AccoRepo, sh *handlers.StorageHandler, orcestrator *CreateAccommodationOrchestrator, t trace.Tracer) *AccoHandler {

	return &AccoHandler{l, r, sh, t, orcestrator}
}

func (ah *AccoHandler) createAccommodation(rw http.ResponseWriter, req *http.Request) {
	ctx, span := ah.tracer.Start(req.Context(), "AccoHandler.createAccommodation") //tracer
	defer span.End()                                                               //tracer

	payload, ok := ctx.Value("payload").(*token.Payload)
	if !ok {
		sendErrorWithMessage(rw, "Authorization token not found", http.StatusInternalServerError)
		return
	}
	hostId, ok := ctx.Value("userId").(string)
	if !ok {
		sendErrorWithMessage(rw, "Authorization token not found", http.StatusInternalServerError)
		return
	}

	if payload.Role == "GUEST" {
		sendErrorWithMessage(rw, "Unauthorized access", http.StatusUnauthorized)
		return
	}
	accommodation := req.Context().Value(KeyProduct{}).(*Accommodation2)
	ah.logger.Println(accommodation)

	accomodations, err := ah.db.GetAllThatAreNotApproved()
	if err != nil {
		ah.logger.Println("Error in GetAllThatAreNotApproved: ", err)
		sendErrorWithMessage(rw, "Cant create accommodation right now please try again", http.StatusInternalServerError)
		return
	}

	for _, value := range accomodations {
		log.Println("Accommodations: ", value)
		err := ah.db.DeleteById(value.ID)
		if err != nil {
			ah.logger.Println("Error in deleting accommodation: ", err)
			sendErrorWithMessage(rw, "Cant create accommodation right now please try again", http.StatusInternalServerError)
			return
		}
	}

	id := randstr.String(24)
	accommodation1 := &Accommodation{
		ID:           id,
		Name:         accommodation.Name,
		Location:     accommodation.Location,
		Amenities:    accommodation.Amenities,
		MinGuests:    accommodation.MinGuests,
		MaxGuests:    accommodation.MaxGuests,
		Username:     accommodation.Username,
		AverageGrade: 0,
		Images:       accommodation.Images,
	}

	accommodation1.Approved = "false"

	// err = ah.db.Insert(accommodation1)
	accommodation.AverageGrade = 0
	err = ah.db.Insert(accommodation1, ctx)
	if err != nil {
		ah.logger.Println("error:1", err.Error())
		if strings.Contains(err.Error(), "duplicate key") {
			sendErrorWithMessage(rw, "accommodation with that name already exists", http.StatusBadRequest)
			return
		}
		sendErrorWithMessage(rw, "", http.StatusBadRequest)
		return
	}

	startDate, err := time.Parse("2006-01-02", accommodation.StartDate)
	if err != nil {
		ah.logger.Println("Error parsing startDate:", err)
		sendErrorWithMessage(rw, "Datas are sent in wrong format", http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse("2006-01-02", accommodation.EndDate)
	if err != nil {
		ah.logger.Println("Error parsing startDate:", err)
		sendErrorWithMessage(rw, "Datas are sent in wrong format", http.StatusBadRequest)
		return
	}

	reservation := &ReservationByAccommodation1{
		AccoId:               id,
		HostId:               hostId,
		NumberPeople:         accommodation.NumberPeople,
		PriceByPeople:        accommodation.PriceByPeople,
		PriceByAccommodation: accommodation.PriceByAccommodation,
		StartDate:            startDate,
		EndDate:              endDate,
	}

	err = ah.CreateSaga(reservation)
	if err != nil {
		err := ah.db.UpdateAccommodation(id) // vidi kako se zove metoda
		if err != nil {
			ah.logger.Println("Error in UpdateAccommodation")
			return
		}
		ah.logger.Println("Error when tryed to create saga in createAccommodation")
		sendErrorWithMessage(rw, "Cant create reservation right now try later", http.StatusInternalServerError)
		return
	}

	sendErrorWithMessage(rw, "Accommodation successfuly created", http.StatusCreated)

	// id := randstr.String(24)
	// accommodation1 := &Accommodation{
	// 	ID:           id,
	// 	Name:         accommodation.Name,
	// 	Location:     accommodation.Location,
	// 	Amenities:    accommodation.Amenities,
	// 	MinGuests:    accommodation.MinGuests,
	// 	MaxGuests:    accommodation.MaxGuests,
	// 	Username:     accommodation.Username,
	// 	AverageGrade: 0,
	// 	Images:       accommodation.Images,
	// }
	// err := ah.db.Insert(accommodation1)
	// if err != nil {
	// 	ah.logger.Println("error:1", err.Error())
	// 	if strings.Contains(err.Error(), "duplicate key") {
	// 		sendErrorWithMessage(rw, "accommodation with that name already exists", http.StatusBadRequest)
	// 		return
	// 	}
	// 	sendErrorWithMessage(rw, "", http.StatusBadRequest)
	// 	return
	// }

	// reservation := &ReservationByAccommodation{
	// 	AccoId:               id,
	// 	NumberPeople:         accommodation.NumberPeople,
	// 	PriceByPeople:        accommodation.PriceByPeople,
	// 	PriceByAccommodation: accommodation.PriceByAccommodation,
	// 	StartDate:            accommodation.StartDate,
	// 	EndDate:              accommodation.EndDate,
	// }

	// log.Println("reservation:", reservation)
	// // create availability periods
	// response, err := ah.db.CreateAvailabilityPeriods(token, reservation)
	// if err != nil {
	// 	ah.logger.Println("Error when try to create availability periods in createAccommodation func: ", err)
	// 	sendErrorWithMessage(rw, "Cant create reservation", http.StatusInternalServerError)
	// 	return
	// }

	// log.Println("response:", response)

	// responseBody, err := ioutil.ReadAll(response.Body)
	// if err != nil {
	// 	ah.logger.Println("Error reading response body:", err)
	// 	sendErrorWithMessage(rw, "Error reading response body", http.StatusInternalServerError)
	// 	return
	// }

	// log.Println("responseBody:", responseBody)

	// defer response.Body.Close()
	// if string(response.StatusCode) == strconv.Itoa(http.StatusCreated) {
	// 	rw.WriteHeader(http.StatusCreated)
	// 	return
	// } else {
	// 	sendErrorWithMessage(rw, string(responseBody), response.StatusCode)
	// 	return
	// }
	// // ah.storageHandler.WriteFileToStorage(rw, req)
	// rw.WriteHeader(http.StatusCreated)

}

func (ah *AccoHandler) GetAccommodationById(w http.ResponseWriter, req *http.Request) {
	ctx, span := ah.tracer.Start(req.Context(), "AccoHandler.GetAccommodationById") //tracer
	defer span.End()

	vars := mux.Vars(req)
	id := vars["id"]

	accommodation, err := ah.db.GetById(id, ctx)
	if err != nil {
		ah.logger.Println(err)
	}

	if accommodation == nil {
		http.Error(w, "Accommodation with given id not found", http.StatusNotFound)
		ah.logger.Printf("Accommodation with id: '%s' not found", id)
		return
	}

	err = accommodation.ToJSON(w)
	if err != nil {
		http.Error(w, "Unable to convert to json", http.StatusInternalServerError)
		ah.logger.Fatal("Unable to convert to json :", err)
		return
	}
}

func (ah *AccoHandler) CreateSaga(reservation *ReservationByAccommodation1) error {
	err := ah.orchestrator.Start(reservation)
	if err != nil {
		error := ah.db.DeleteById(reservation.AccoId)
		if error != nil {
			log.Println("Error when try to delete accommodation by id in CreateSaga func")
			return err
		}
		return err
	}

	return nil
}

func (ah *AccoHandler) GetAllAccommodationsByUsername(w http.ResponseWriter, req *http.Request) {
	ctx, span := ah.tracer.Start(req.Context(), "AccoHandler.GetAllAccommodationsByUsername") //tracer
	defer span.End()

	vars := mux.Vars(req)
	username := vars["username"]

	accommodations, err := ah.db.GetAllByUsername(username, ctx)
	if err != nil {
		ah.logger.Print("Database exception: ", err)
	}

	if accommodations == nil {
		http.Error(w, "Accommodations with given username not found", http.StatusNotFound)
		ah.logger.Printf("Accommodations with username: '%s' not found", username)
		return
	}

	err = accommodations.ToJSON(w)
	if err != nil {
		http.Error(w, "Unable to convert to json", http.StatusInternalServerError)
		ah.logger.Fatal("Unable to convert to json :", err)
		return
	}
}

func (ah *AccoHandler) GetAllAccommodationsById(w http.ResponseWriter, req *http.Request) {
	ctx, span := ah.tracer.Start(req.Context(), "AccoHandler.GetAllAccommodationsById") //tracer
	defer span.End()

	vars := mux.Vars(req)
	id := vars["id"]
	ah.logger.Println(id)
	accommodations, err := ah.db.GetAllById(id, ctx)
	if err != nil {
		ah.logger.Print("Database exception: ", err)
	}

	if accommodations == nil {
		http.Error(w, "Accommodations with given id not found", http.StatusNotFound)
		ah.logger.Printf("Accommodations with id: '%s' not found", id)
		return
	}

	err = accommodations.ToJSON(w)
	if err != nil {
		http.Error(w, "Unable to convert to json", http.StatusInternalServerError)
		ah.logger.Fatal("Unable to convert to json :", err)
		return
	}
}

func (ah *AccoHandler) UpdateAccommodationGrade(res http.ResponseWriter, req *http.Request) {
	ctx, span := ah.tracer.Start(req.Context(), "AccoHandler.UpdateAccommodationGrade") //tracer
	defer span.End()

	vars := mux.Vars(req)
	id := vars["id"]

	payload, ok := ctx.Value("payload").(*token.Payload)
	if !ok {
		sendErrorWithMessage(res, "Authorization token not found", http.StatusInternalServerError)
		return
	}

	if payload.Role == "HOST" {
		sendErrorWithMessage(res, "Unauthorized access", http.StatusUnauthorized)
		return
	}

	userId, ok := req.Context().Value("userId").(string)
	if !ok {
		ah.logger.Println("Error retrieving userId from context")
		sendErrorWithMessage1(res, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ah.logger.Println("UserId:", userId)
	ah.logger.Println("AccommodationGradeId:", id)

	err := ah.db.DeleteAccommodationGrade(userId, id, ctx)
	if err != nil {
		ah.logger.Println("Error2:", err)
		sendErrorWithMessage(res, err.Error(), http.StatusInternalServerError)
		return
	}

	sendErrorWithMessage(res, "Accommodation grade succesfully deleted", http.StatusOK)
}

func (ah *AccoHandler) GetAllRecommended(w http.ResponseWriter, req *http.Request) {

	accoIds := &ReqList{}
	err := accoIds.FromJSON(req.Body)
	if err != nil {
		http.Error(w, "Unable to decode json", http.StatusBadRequest)
		ah.logger.Fatal(err)
		return
	}

	ah.logger.Println(accoIds)

	recommended, err := ah.db.GetAllRecommended(accoIds)
	if err != nil {
		ah.logger.Print("Database exception: ", err)
	}

	if recommended == nil {
		http.Error(w, "Accommodations with given ids not found", http.StatusNotFound)
		ah.logger.Printf("Accommodations with given ids not found")
		return
	}

	err = recommended.ToJSON(w)
	if err != nil {
		http.Error(w, "Unable to convert to json", http.StatusInternalServerError)
		ah.logger.Fatal("Unable to convert to json :", err)
		return
	}
}

func (ah *AccoHandler) GetAllAccommodationsByLocation(w http.ResponseWriter, req *http.Request) {
	ctx, span := ah.tracer.Start(req.Context(), "AccoHandler.GetAllAccommodationsByLocation") //tracer
	defer span.End()

	vars := mux.Vars(req)
	locations := vars["locations"]
	accommodations, err := ah.db.GetAllByLocation(locations, ctx)
	if err != nil {
		ah.logger.Print("Database exception: ", err)
	}

	if accommodations == nil {
		http.Error(w, "Accommodations with given location not found", http.StatusNotFound)
		ah.logger.Printf("Accommodations with location: '%s' not found", locations)
		return
	}

	err = accommodations.ToJSON(w)
	if err != nil {
		http.Error(w, "Unable to convert to json", http.StatusInternalServerError)
		ah.logger.Fatal("Unable to convert to json :", err)
		return
	}
}

type ReservationDateByDateGet struct {
	AccoId string `json:"acco_id"`
}

func (ah *AccoHandler) GetAllAccommodationsByDate(w http.ResponseWriter, req *http.Request) {
	ctx, span := ah.tracer.Start(req.Context(), "AccoHandler.GetAllAccommodationsByDate") //tracer
	defer span.End()

	vars := mux.Vars(req)
	startDate := vars["startDate"]
	endDate := vars["endDate"]
	var accommodations Accommodations

	accoIdListResponse, err := ah.db.GetAllAccoFromReservationServiceByDate(startDate, endDate, ctx)
	if err != nil {
		ah.logger.Print("Database exception: ", err)
		http.Error(w, "Error fetching accommodation IDs", http.StatusInternalServerError)
		return
	}
	log.Println(accoIdListResponse)

	var accommodationIdList []ReservationDateByDateGet
	if err := json.NewDecoder(accoIdListResponse.Body).Decode(&accommodationIdList); err != nil {
		log.Println("Decoding error, da ne kazem glup si... :", err)
	}
	log.Println(accommodationIdList)

	for _, accoID := range accommodationIdList {
		log.Println(accoID)
		accommodation, err := ah.db.GetById(accoID.AccoId, ctx)
		if err != nil {
			ah.logger.Printf("Error fetching accommodation with ID %d: %v", accoID, err)
			continue
		}
		accommodations = append(accommodations, accommodation)
	}

	log.Println(accommodations)
	if accommodations == nil {
		http.Error(w, "Accommodations with given date not found", http.StatusNotFound)
		ah.logger.Printf("Accommodations with date: '%s' '%s' not found", startDate, endDate)
		return
	}

	err = accommodations.ToJSON(w)
	if err != nil {
		http.Error(w, "Unable to convert to json", http.StatusInternalServerError)
		ah.logger.Fatal("Unable to convert to json :", err)
		return
	}
}

func (ah *AccoHandler) DeleteAccommodationGrade(res http.ResponseWriter, req *http.Request) {
	ctx, span := ah.tracer.Start(req.Context(), "AccoHandler.DeleteAccommodationGrade") //tracer
	defer span.End()

	vars := mux.Vars(req)
	id := vars["id"]

	payload, ok := ctx.Value("payload").(*token.Payload)
	if !ok {
		sendErrorWithMessage(res, "Authorization token not found", http.StatusInternalServerError)
		return
	}

	if payload.Role == "HOST" {
		sendErrorWithMessage(res, "Unauthorized access", http.StatusUnauthorized)
		return
	}

	userId, ok := req.Context().Value("userId").(string)
	if !ok {
		ah.logger.Println("Error retrieving userId from context")
		sendErrorWithMessage1(res, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ah.logger.Println("UserId:", userId)
	ah.logger.Println("AccommodationGradeId:", id)

	err := ah.db.DeleteAccommodationGrade(userId, id, ctx)
	if err != nil {
		ah.logger.Println("Error2:", err)
		sendErrorWithMessage(res, err.Error(), http.StatusInternalServerError)
		return
	}

	sendErrorWithMessage(res, "Accommodation grade succesfully deleted", http.StatusOK)
}

func (ah *AccoHandler) DeleteAccommodation(res http.ResponseWriter, req *http.Request) {
	ctx, span := ah.tracer.Start(req.Context(), "AccoHandler.DeleteAccommodation") //tracer
	defer span.End()

	vars := mux.Vars(req)
	username := vars["username"]

	payload, ok := ctx.Value("payload").(*token.Payload)
	if !ok {
		sendErrorWithMessage(res, "Authorization token not found", http.StatusInternalServerError)
		return
	}

	if payload.Role == "GUEST" {
		sendErrorWithMessage(res, "Unauthorized access", http.StatusUnauthorized)
		return
	}

	err := ah.db.Delete(username, ctx)
	if err != nil {
		ah.logger.Println("Error when tried to delete accommodation:", err)
		sendErrorWithMessage1(res, "Error when tried to delete accommodation", http.StatusInternalServerError)
		return
	}
	sendErrorWithMessage1(res, "User succesfully deleted", http.StatusOK)
}

func (ah *AccoHandler) MiddlewareRoleCheck00(client *http.Client, breaker *gobreaker.CircuitBreaker, tokenMaker token.Maker) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()

			reqURL := "https://auth-service:8000/api/users/auth"

			authorizationHeader := r.Header.Get("authorization")
			fields := strings.Fields(authorizationHeader)

			if len(fields) == 0 {
				sendErrorWithMessage(w, "", http.StatusUnauthorized)
				return
			}

			accessToken := fields[1]

			var token ReqToken
			token.Token = accessToken

			jsonToken, _ := json.Marshal(token)

			cbResp, err := breaker.Execute(func() (interface{}, error) {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, bytes.NewBuffer(jsonToken))
				if err != nil {
					return nil, err
				}

				tr := http.DefaultTransport.(*http.Transport).Clone()
				tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

				client := http.Client{Transport: tr}
				return client.Do(req)
			})
			if err != nil {
				ah.logger.Println(err)
				sendErrorWithMessage(w, "Service is not working", http.StatusInternalServerError)
				return
			}

			resp := cbResp.(*http.Response)
			resBody, err := io.ReadAll(resp.Body)
			ah.logger.Println("User Id:", string(resBody))
			if resp.StatusCode != http.StatusOK {
				ah.logger.Println("Error in auth response " + strconv.Itoa(resp.StatusCode))
				ah.logger.Println("status " + resp.Status)
				sendErrorWithMessage(w, "Lavor", resp.StatusCode)
				return
			}
			payload, err := tokenMaker.VerifyToken(accessToken)
			if err != nil {
				// If the token verification fails, return an error
				writeError(w, http.StatusUnauthorized, err)
				return
			}

			ctx = context.WithValue(ctx, "payload", payload)

			userID := string(resBody)
			ctx = context.WithValue(ctx, "userId", userID)
			ctx = context.WithValue(ctx, "accessToken", accessToken)

			newReq, err := http.NewRequestWithContext(ctx, r.Method, r.URL.String(), r.Body)
			if err != nil {
				ah.logger.Println("Error creating new request:", err)
				sendErrorWithMessage(w, "Error creating new request:"+err.Error(), http.StatusInternalServerError)
				return
			}
			newReq.Header = r.Header

			newReq.Header.Set("Content-Type", "application/json")

			ah.logger.Println("Token:", token)
			ah.logger.Println("AccessToken:", accessToken)

			next.ServeHTTP(w, newReq)
		})
	}
}

func writeError(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
}
func (ah *AccoHandler) GetAllAccommodationGrades(res http.ResponseWriter, req *http.Request) {
	//ctx, span := ah.tracer.Start(req.Context(), "AccoHandler.GetAllAccommodationGrades") //tracer
	//defer span.End()

	vars := mux.Vars(req)
	accommodationId := vars["id"]

	ctx := req.Context()

	userId, ok := ctx.Value("userId").(string)
	if !ok {
		sendErrorWithMessage(res, "User not found", http.StatusInternalServerError)
		return
	}

	ah.logger.Println("UserId: ", userId)
	payload, ok := ctx.Value("payload").(*token.Payload)
	if !ok {
		sendErrorWithMessage(res, "Payload not found", http.StatusInternalServerError)
		return
	}

	ah.logger.Println("Payload: ", payload)

	accommodationGrades, err := ah.db.GetAllAccommodationGrades(accommodationId)
	if err != nil {
		ah.logger.Println("Error1:", err.Error())
		sendErrorWithMessage(res, "Error in GetAllAccommodationGrades method", http.StatusInternalServerError)
		return
	}

	err = accommodationGrades.ToJSON(res)
	if err != nil {
		ah.logger.Println("Unable to convert to json :", err)
		sendErrorWithMessage(res, "Unable to convert to json", http.StatusInternalServerError)
		return
	}
}

func (ah *AccoHandler) GetAllAccommodationsByNoGuests(w http.ResponseWriter, req *http.Request) {
	ctx, span := ah.tracer.Start(req.Context(), "AccoHandler.GetAllAccommodationsByNoGuests") //tracer
	defer span.End()

	vars := mux.Vars(req)
	noGuests := vars["noGuests"]

	accommodations, err := ah.db.GetAllByNoGuests(noGuests, ctx)
	if err != nil {
		ah.logger.Print("Database exception: ", err)
	}

	if accommodations == nil {
		http.Error(w, "Accommodations with given noGuests not found", http.StatusNotFound)
		ah.logger.Printf("Accommodations with noGuests: '%s' not found", noGuests)
		return
	}

	err = accommodations.ToJSON(w)
	if err != nil {
		http.Error(w, "Unable to convert to json", http.StatusInternalServerError)
		ah.logger.Fatal("Unable to convert to json :", err)
		return
	}
}

func (ah *AccoHandler) getAllAccommodations(rw http.ResponseWriter, req *http.Request) {
	ctx, span := ah.tracer.Start(req.Context(), "AccoHandler.getAllAccommodations") //tracer
	defer span.End()

	accommodations, err := ah.db.GetAll(ctx)
	if err != nil {
		ah.logger.Print("Database exception: ", err)
	}

	if accommodations == nil {
		return
	}

	err = accommodations.ToJSON(rw)
	if err != nil {
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		ah.logger.Fatal("Unable to convert to json :", err)
		return
	}
}

func (ah *AccoHandler) MiddlewareAccommodationDeserialization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		accommodation := &Accommodation2{}
		err := accommodation.FromJSON(h.Body)
		if err != nil {
			http.Error(rw, "Unable to decode JSON", http.StatusBadRequest)
			ah.logger.Fatal(err)
			return
		}

		ctx := context.WithValue(h.Context(), KeyProduct{}, accommodation)
		h = h.WithContext(ctx)

		next.ServeHTTP(rw, h)
	})
}

func (ah *AccoHandler) MiddlewareRoleCheck(client *http.Client, breaker *gobreaker.CircuitBreaker, tokenMaker token.Maker) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()
			reqURL := "https://auth-service:8000/api/users/auth"

			authorizationHeader := r.Header.Get("authorization")
			fields := strings.Fields(authorizationHeader)
			if len(fields) == 0 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			ah.logger.Println(fields)

			accessToken := fields[1]

			var token ReqToken
			token.Token = accessToken

			jsonToken, _ := json.Marshal(token)

			cbResp, err := breaker.Execute(func() (interface{}, error) {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, bytes.NewBuffer(jsonToken))
				if err != nil {
					return nil, err
				}
				tr := http.DefaultTransport.(*http.Transport).Clone()
				tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

				client := http.Client{Transport: tr}
				return client.Do(req)
			})
			if err != nil {
				ah.logger.Println(err)
				sendErrorWithMessage(w, "Service is not working", http.StatusInternalServerError)
				return
			}

			resp := cbResp.(*http.Response)
			resBody, err := io.ReadAll(resp.Body)
			ah.logger.Println(string(resBody))
			if resp.StatusCode != http.StatusOK {
				ah.logger.Println("Error in auth response " + strconv.Itoa(resp.StatusCode))
				ah.logger.Println("status " + resp.Status)
				w.WriteHeader(resp.StatusCode)
				return
			}
			ah.logger.Println(resp)

			ah.logger.Println("Pre payloda")
			// var accessToken = fields[1]
			payload, err := tokenMaker.VerifyToken(accessToken)
			if err != nil {
				// If the token verification fails, return an error
				writeError(w, http.StatusUnauthorized, err)
				return
			}
			ah.logger.Println("Payload in middleware: ", payload)

			ctx = context.WithValue(ctx, "payload", payload)

			log.Println(accessToken)
			userID := string(resBody)
			ctx = context.WithValue(ctx, "userId", userID)

			ctx = context.WithValue(ctx, AccessTokenKey, accessToken)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

func (ah *AccoHandler) GradeAccommodation(res http.ResponseWriter, req *http.Request) {
	ctx, span := ah.tracer.Start(req.Context(), "AccoHandler.GradeAccommodation") //tracer
	defer span.End()

	payload, ok := ctx.Value("payload").(*token.Payload)
	if !ok {
		sendErrorWithMessage(res, "Authorization token not found", http.StatusInternalServerError)
		return
	}

	if payload.Role == "HOST" {
		sendErrorWithMessage(res, "Unauthorized access", http.StatusUnauthorized)
		return
	}

	ah.logger.Println("Request Body:", req.Body)
	accommodationGrade, err := decodeAccommodatioGradeBody(req.Body)
	if err != nil {
		ah.logger.Println("Cant decode body")
		sendErrorWithMessage(res, err.Error(), http.StatusBadRequest)
		return
	}

	userId, ok := req.Context().Value("userId").(string)
	if !ok {
		ah.logger.Println("Error retrieving userId from context")
		sendErrorWithMessage(res, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	currentTime := time.Now()
	formattedTime := currentTime.Format("2006-01-02 15:04:05")
	accommodationGrade.CreatedAt = formattedTime
	accommodationGrade.ID = randstr.String(20)
	accommodationGrade.UserId = strings.Trim(userId, `"`)
	ah.logger.Println("HostGrade:", accommodationGrade)

	tocken, ok := req.Context().Value("accessToken").(string)
	if !ok {
		ah.logger.Println("Error retrieving tocken from context", err)
		sendErrorWithMessage(res, "", http.StatusInternalServerError)
		return
	}

	err = ah.db.CreateGrade(accommodationGrade, tocken, ctx)
	if err != nil {
		ah.logger.Println("Error in inserting accommodation grade", err)
		sendErrorWithMessage(res, err.Error(), http.StatusInternalServerError)
		return
	}

	sendErrorWithMessage(res, "Accommodation succesfuly created", http.StatusCreated)
}

func (ah *AccoHandler) MiddlewareContentTypeSet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		ah.logger.Println("Method [", h.Method, "] - Hit path :", h.URL.Path)

		rw.Header().Add("Content-Type", "application/json")

		next.ServeHTTP(rw, h)
	})
}

func decodeBody(r io.Reader) (*Accommodation, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var rt Accommodation
	if err := dec.Decode(&rt); err != nil {
		return nil, err
	}

	err := ValidateAccommodation(&rt)
	if err != nil {
		return nil, err
	}

	return &rt, nil
}

func renderJSON(w http.ResponseWriter, v interface{}) {
	js, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func decodeAcco2Body(r io.Reader) (*Accommodation2, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var rt Accommodation2
	if err := dec.Decode(&rt); err != nil {
		return nil, err
	}
	return &rt, nil
}

func decodeAccommodatioGradeBody(r io.Reader) (*AccommodationGrade, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var rt AccommodationGrade
	if err := dec.Decode(&rt); err != nil {
		return nil, err
	}

	err := ValidateAccommodationGrade(&rt)
	if err != nil {
		return nil, err
	}

	return &rt, nil
}

func sendErrorWithMessage(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errorResponse := map[string]string{"message": message}
	json.NewEncoder(w).Encode(errorResponse)
}

func sendErrorWithMessage1(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(message))
	w.WriteHeader(statusCode)
}

func (ah *AccoHandler) ExtractTraceInfoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
