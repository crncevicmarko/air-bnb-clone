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
)

type KeyProduct struct{}

type reservationHandler struct {
	logger *log.Logger
	repo   *ReservationRepo
	tracer trace.Tracer
}

func NewReservationHandler(l *log.Logger, r *ReservationRepo, t trace.Tracer) *reservationHandler {
	return &reservationHandler{l, r, t}
}

//type AccoHandler struct {
//	logger *log.Logger
//	db     *AccoRepo
//}
//func NewAccoHandler(l *log.Logger, r *AccoRepo) *AccoHandler {
//
//	return &AccoHandler{l, r}
//}

func (rh *reservationHandler) GetAllReservationIds(res http.ResponseWriter, req *http.Request) {
	//ctx, span := rh.tracer.Start(req.Context(), "reservationHandler.createNotification") //tracer
	//defer span.End()                                                                     //tracer

	reservationIds, err := rh.repo.GetDistinctIds("reservation_id", "reservations_by_user")
	if err != nil {
		rh.logger.Print("Database exception: ", err)
	}

	if reservationIds == nil {
		return
	}

	rh.logger.Println(reservationIds)

	e := json.NewEncoder(res)
	err = e.Encode(reservationIds)
	if err != nil {
		http.Error(res, "Unable to convert to json", http.StatusInternalServerError)
		rh.logger.Fatal("Unable to convert to json :", err)
		return
	}
}

func (rh *reservationHandler) GetReservationDatesByAccommodationId(res http.ResponseWriter, req *http.Request) {
	ctx, span := rh.tracer.Start(req.Context(), "reservationHandler.GetReservationDatesByAccommodationId") //tracer
	defer span.End()

	vars := mux.Vars(req)
	accoId := vars["id"]

	reservationDatesByAccomodationId, err := rh.repo.GetReservationsDatesByAccommodationId(accoId, ctx)
	if err != nil {
		rh.logger.Print("Database exception: ", err)
		sendErrorWithMessage(res, "Error when getting reservation dates", http.StatusBadRequest)
		return
	}

	if reservationDatesByAccomodationId == nil {
		return
	}

	err = reservationDatesByAccomodationId.ToJSON(res)
	if err != nil {
		sendErrorWithMessage(res, "Unable to convert to json", http.StatusInternalServerError)
		rh.logger.Fatal("Unable to convert to json :", err)
		return
	}
}

func (rh *reservationHandler) GetAllReservationsByAccommodationId(res http.ResponseWriter, req *http.Request) {
	ctx, span := rh.tracer.Start(req.Context(), "reservationHandler.GetAllReservationsByAccommodationId") //tracer
	defer span.End()

	vars := mux.Vars(req)
	accoId := vars["id"]

	reservationsByAcco, err := rh.repo.GetReservationsByAcco(accoId, ctx)
	if err != nil {
		rh.logger.Print("Database exception: ", err)
		sendErrorWithMessage(res, "Error when getting reservations", http.StatusBadRequest)
		return
	}

	if reservationsByAcco == nil {
		sendErrorWithMessage(res, "There is no reservation for that accommodation", http.StatusBadRequest)
		return
	}

	err = reservationsByAcco.ToJSON(res)
	if err != nil {
		sendErrorWithMessage(res, "Unable to convert to json", http.StatusInternalServerError)
		rh.logger.Fatal("Unable to convert to json :", err)
		return
	}
}

func (rh *reservationHandler) GetAllReservationsDatesByDate(res http.ResponseWriter, req *http.Request) {
	ctx, span := rh.tracer.Start(req.Context(), "reservationHandler.GetAllReservationsDatesByDate") //tracer
	defer span.End()

	vars := mux.Vars(req)
	startDate := vars["startDate"]
	endDate := vars["endDate"]

	reservationsByAcco, err := rh.repo.GetReservationsDatesByDate(startDate, endDate, ctx)
	if err != nil {
		rh.logger.Print("Database exception: ", err)
		sendErrorWithMessage(res, "Error when getting reservations", http.StatusBadRequest)
		return
	}

	if reservationsByAcco == nil {
		return
	}

	err = reservationsByAcco.ToJSON(res)
	//log.Println("ALO BRE", res)
	//log.Println("ISPIS ONOGA STO VRACA RES SERVIS", reservationsByAcco.ToJSON(res))
	if err != nil {
		sendErrorWithMessage(res, "Unable to convert to json", http.StatusInternalServerError)
		rh.logger.Fatal("Unable to convert to json :", err)
		return
	}
}

func (rh *reservationHandler) getAllReservationsByUser(res http.ResponseWriter, req *http.Request) {
	ctx, span := rh.tracer.Start(req.Context(), "reservationHandler.getAllReservationsByUser") //tracer
	defer span.End()

	//vars := mux.Vars(req)
	////userId := vars["id"]
	requestId, err := decodeIdBody(req.Body)
	reservationsByUser, err := rh.repo.GetReservationsByUser(requestId.UserId, ctx)
	// vars := mux.Vars(req)
	// userId := vars["id"]
	// reservationsByUser, err := rh.repo.GetReservationsByUser(userId)
	if err != nil {
		rh.logger.Println("Database exception: ", err)
		sendErrorWithMessage(res, "Error in getting reservation", http.StatusBadRequest)
		return
	}

	if reservationsByUser == nil {
		return
	}

	err = reservationsByUser.ToJSON(res)
	if err != nil {
		rh.logger.Println("Unable to convert to json :", err)
		sendErrorWithMessage(res, "Unable to convert to json", http.StatusBadRequest)
		return
	}
}

func (rh *reservationHandler) GetAllReservationsByUserId(res http.ResponseWriter, req *http.Request) {
	ctx, span := rh.tracer.Start(req.Context(), "reservationHandler.GetAllReservationsByUserId") //tracer
	defer span.End()

	rh.logger.Println("Request Body: ", req.Body)
	requestId, err := decodeIdBody(req.Body)
	if err != nil {
		rh.logger.Println("Cant decode body")
		sendErrorWithMessage(res, "Cant decode body", http.StatusBadRequest)
		return
	}

	reservationsByUser, err := rh.repo.GetReservationsByUser(requestId.UserId, ctx)
	if err != nil {
		rh.logger.Println("Database exception: ", err)
		sendErrorWithMessage(res, "Error in getting reservation", http.StatusBadRequest)
		return
	}

	if reservationsByUser == nil {
		return
	}

	err = reservationsByUser.ToJSON(res)
	if err != nil {
		rh.logger.Println("Unable to convert to json :", err)
		sendErrorWithMessage(res, "Unable to convert to json", http.StatusBadRequest)
		return
	}
}

func (rh *reservationHandler) GetAllReservationsDatesByHostId(res http.ResponseWriter, req *http.Request) {
	ctx, span := rh.tracer.Start(req.Context(), "reservationHandler.GetAllReservationsDatesByHostId") //tracer
	defer span.End()

	vars := mux.Vars(req)
	hostId := vars["id"]
	rh.logger.Println("Usli u GetReservationsDatesByHostId met")

	reservationsDatesByHostId, err := rh.repo.GetReservationsDatesByHostId(hostId, ctx)
	if err != nil {
		rh.logger.Print("Database exception: ", err)
		sendErrorWithMessage1(res, "Error when getting reservations", http.StatusBadRequest)
		return
	}

	if reservationsDatesByHostId == nil {
		sendErrorWithMessage1(res, "There is no availability dates for that accommodation", http.StatusBadRequest)
		return
	}

	var activeReservations = false
	var counterIsThereAnyReservations = 0
	for _, element := range reservationsDatesByHostId {
		rh.logger.Println("Reservation:", element)
		reservationsByAccomodationIdFromReservationsByUserTable, err := rh.repo.GetReservationsDatesByAccommodationId(element.AccoId, ctx)
		if err != nil {
			rh.logger.Println("Cant get reservations by accommodation:", err)
			sendErrorWithMessage1(res, "Cant get reservations by accommodationId for host", http.StatusInternalServerError)
			return
		}

		var isDatePassedd = false
		if len(reservationsByAccomodationIdFromReservationsByUserTable) != 0 {
			counterIsThereAnyReservations++
			for _, element1 := range reservationsByAccomodationIdFromReservationsByUserTable {
				rh.logger.Println("Reservation by AccoId:", element1)
				response, err := rh.repo.isDatePassed(element1.EndAccomodationDate)
				if err != nil {
					counterIsThereAnyReservations = 0
					rh.logger.Println("Error in isDatePassed metod:", err)
					sendErrorWithMessage1(res, "Error in isDatePassed metod", http.StatusInternalServerError)
					return
				}
				rh.logger.Println("Response:", response)
				isDatePassedd = response
				if !response {
					break
				}
			}
		}

		activeReservations = isDatePassedd
		rh.logger.Println("Active:", activeReservations)
		if !isDatePassedd {
			break
		}

	}

	if counterIsThereAnyReservations == 0 {
		rh.logger.Println("No reservations")
		sendErrorWithMessage1(res, "There is not reservations for hosts accommodations", http.StatusBadRequest)
		return
	}
	rh.logger.Println("Active2:", activeReservations)
	if !activeReservations {
		counterIsThereAnyReservations = 0
		sendErrorWithMessage1(res, "There is active reservations for accommodations of this host", http.StatusOK)
		return
	} else {
		counterIsThereAnyReservations = 0
		sendErrorWithMessage1(res, "There is no active reservations for accommodations of this host", http.StatusOK)
		return
	}
}

func (rh *reservationHandler) GetAllReservationsForUserIdByHostId(res http.ResponseWriter, req *http.Request) {
	ctx, span := rh.tracer.Start(req.Context(), "reservationHandler.GetAllReservationsForUserIdByHostId") //tracer
	defer span.End()

	vars := mux.Vars(req)
	userId := vars["userId"]
	hostId := vars["hostId"]
	rh.logger.Println("Usli u GetReservationsDatesByHostId met")

	reservationsDatesByHostId, err := rh.repo.GetReservationsDatesByHostId(hostId, ctx)
	if err != nil {
		rh.logger.Print("Database exception: ", err)
		sendErrorWithMessage1(res, "Error when getting reservations", http.StatusBadRequest)
		return
	}

	if reservationsDatesByHostId == nil {
		sendErrorWithMessage1(res, "There is no accommodation for that host", http.StatusBadRequest)
		return
	}

	var isThereSomePreviousReservations = false
	var counterIsThereAnyReservations = 0
	for _, element := range reservationsDatesByHostId {
		rh.logger.Println("Reservation:", element)
		reservationsByUserId, err := rh.repo.GetReservationsByUser(userId, ctx)
		if err != nil {
			rh.logger.Println("Cant get reservations by user:", err)
			sendErrorWithMessage1(res, "Cant get reservations by userId for userId", http.StatusInternalServerError)
			return
		}

		if len(reservationsByUserId) != 0 {
			counterIsThereAnyReservations++
			for _, element1 := range reservationsByUserId {
				rh.logger.Println("Reservation by AccoId:", element1)
				if element.AccoId == element1.AccoId {
					isThereSomePreviousReservations = true
					break
				}
			}
		}
	}
	if counterIsThereAnyReservations == 0 {
		rh.logger.Println("No reservations")
		sendErrorWithMessage1(res, "There is no reservations for hosts accommodations", http.StatusBadRequest)
		return
	}

	if isThereSomePreviousReservations == true {
		counterIsThereAnyReservations = 0
		rh.logger.Println("There is some reservtions for this user")
		sendErrorWithMessage1(res, "There is some reservtions for this user", http.StatusOK)
		return
	}
	counterIsThereAnyReservations = 0
	sendErrorWithMessage1(res, "There is no reservations for this user", http.StatusBadRequest)
}

func (rh *reservationHandler) IsHostProminent(res http.ResponseWriter, req *http.Request) {
	ctx, span := rh.tracer.Start(req.Context(), "reservationHandler.IsHostProminent") //tracer
	defer span.End()

	vars := mux.Vars(req)
	hostId := vars["id"]
	rh.logger.Println("Entered IsHostProminent func")

	reservationsDatesByHostId, err := rh.repo.GetReservationsDatesByHostId(hostId, ctx)
	if err != nil {
		rh.logger.Print("Database exception: ", err)
		sendErrorWithMessage1(res, "Error when getting reservations", http.StatusBadRequest)
		return
	}

	if reservationsDatesByHostId == nil {
		rh.logger.Print("There is no accommodation for that host in func IsHostProminent")
		sendErrorWithMessage1(res, "There is no accommodation for that host", http.StatusBadRequest)
		return
	}

	var counterIsThereAnyReservations = 0
	var numOfReservationsForAccommodationOfThatHost = 0
	var allDates []time.Time
	processedAccommodations := make(map[string]bool) // Keep track of processed accommodations

	for _, element := range reservationsDatesByHostId {
		if processedAccommodations[element.AccoId] {
			continue
		}

		processedAccommodations[element.AccoId] = true // Mark the accommodation as processed

		rh.logger.Println("Reservation:", element)
		reservationsByAccId, err := rh.repo.GetReservationsDatesByAccommodationId(element.AccoId, ctx)
		if err != nil {
			rh.logger.Println("Cant get reservations by accommodationId:", err)
			sendErrorWithMessage1(res, "Cant get reservations by accommodationId", http.StatusInternalServerError)
			return
		}

		if len(reservationsByAccId) != 0 {
			counterIsThereAnyReservations++
			for _, element1 := range reservationsByAccId {
				numOfReservationsForAccommodationOfThatHost++
				rh.logger.Println("Reservation by AccoId:", element1)
				start := element1.BeginAccomodationDate
				end := element1.EndAccomodationDate

				for current := start; current.Before(end) || current.Equal(end); current = current.Add(24 * time.Hour) {
					allDates = append(allDates, current)
				}
			}
		}
	}

	numOfDates := len(allDates)
	rh.logger.Println("Number of dates: %d\n", numOfDates)

	if counterIsThereAnyReservations == 0 {
		rh.logger.Println("No reservations")
		sendErrorWithMessage1(res, "false", http.StatusBadRequest)
		return
	}

	if numOfReservationsForAccommodationOfThatHost >= 5 || numOfDates == 50 {
		rh.logger.Println("There is min 5 reservations for accommodations of this host")
		sendErrorWithMessage1(res, "true", http.StatusOK)
		return
	}

	rh.logger.Println("Number:", numOfReservationsForAccommodationOfThatHost)
	sendErrorWithMessage1(res, "false", http.StatusOK)
}

func (rh *reservationHandler) CreateReservationDateForDate(res http.ResponseWriter, req *http.Request) {
	ctx, span := rh.tracer.Start(req.Context(), "reservationHandler.CreateReservationDateForDate") //tracer
	defer span.End()

	reservationDate, err := decodeBody(req.Body)
	if err != nil {
		rh.logger.Println("Error in decoding body")
		sendErrorWithMessage(res, "Error in decoding body", http.StatusBadRequest)
		return
	}

	err = rh.repo.InsertReservationDateByDate(reservationDate, ctx)
	if err != nil {
		rh.logger.Print("Database exception: ", err)
		sendErrorWithMessage(res, err.Error(), http.StatusBadRequest)
		return
	}
	res.WriteHeader(http.StatusCreated)
}

// func (rh *reservationHandler) CreateReservationDateForAccommodation(res http.ResponseWriter, req *http.Request) {
// 	reservationDate, err := decodeBody(req.Body)
// 	if err != nil {
// 		log.Println("Error in decoding body")
// 		sendErrorWithMessage(res, "Error in decoding body", http.StatusBadRequest)
// 		return
// 	}

// 	err = rh.repo.InsertReservationDateForAccomodation(reservationDate)
// 	if err != nil {
// 		rh.logger.Print("Database exception: ", err)
// 		sendErrorWithMessage(res, err.Error(), http.StatusBadRequest)
// 		return
// 	}
// 	res.WriteHeader(http.StatusCreated)
// }

func (rh *reservationHandler) CreateReservationForAcco(res http.ResponseWriter, req *http.Request) {
	log.Println("CreateReservationForAcco")
	//ctx, span := rh.tracer.Start(req.Context(), "reservationHandler.CreateReservationForAcco") //tracer
	//defer span.End()
	reservation, err := decodeReservationBody(req.Body)
	if err != nil {
		rh.logger.Println("Error in decoding body")
		sendErrorWithMessage(res, "Error in decoding body", http.StatusBadRequest)
		return
	}
	err = rh.repo.InsertReservationByAcco(reservation)
	if err != nil {
		rh.logger.Print("Database exception: ", err)
		sendErrorWithMessage(res, "Cant create reservation", http.StatusBadRequest)
		return
	}
	res.WriteHeader(http.StatusCreated)
}

func (rh *reservationHandler) CreateReservationForUser(res http.ResponseWriter, req *http.Request) {
	ctx, span := rh.tracer.Start(req.Context(), "reservationHandler.CreateReservationForUser") //tracer
	defer span.End()

	reservationUser, err := decodeReservationByUserBody(req.Body)
	if err != nil {
		sendErrorWithMessage(res, "Cant decode body", http.StatusBadRequest)
		return
	}

	log.Println("Reservation:", reservationUser)
	err = rh.repo.InsertReservationByUser(reservationUser, ctx)
	if err != nil {
		rh.logger.Print("Database exception: ", err)
		if strings.Contains(err.Error(), "Cant reserve in past") {
			sendErrorWithMessage(res, "Cant reserve in past", http.StatusBadRequest)
			return
		} else if strings.Contains(err.Error(), "Dates are already reserved for that accommodation") {
			sendErrorWithMessage(res, "Dates are already reserved for that accommodation", http.StatusBadRequest)
		} else {
			sendErrorWithMessage(res, "Cant create reservation", http.StatusBadRequest)
		}
		return
	}
	res.WriteHeader(http.StatusCreated)
}

func (rh *reservationHandler) UpdateReservationByUser(res http.ResponseWriter, req *http.Request) {
	ctx, span := rh.tracer.Start(req.Context(), "reservationHandler.UpdateReservationByUser") //tracer
	defer span.End()

	reservation, err := decodeReservationByUserBody(req.Body)
	if err != nil {
		rh.logger.Println(err)
		sendErrorWithMessage(res, "Cant decode body", http.StatusBadRequest)
		return
	}

	err = rh.repo.UpdateReservationByUser(reservation, ctx)
	if err != nil {
		rh.logger.Print("Database exception: ", err)
		if strings.Contains(err.Error(), "Cant find reservation") {
			sendErrorWithMessage(res, "There is no reservation for that date range", http.StatusBadRequest)
		} else if strings.Contains(err.Error(), "Reservation cant be canceled") {
			sendErrorWithMessage(res, "Reservation cant be canceled", http.StatusBadRequest)
		} else {
			sendErrorWithMessage(res, "Cant update reservation", http.StatusBadRequest)
		}
		return
	}
	res.WriteHeader(http.StatusOK)
}

func (rh *reservationHandler) UpdateReservationByAcco(res http.ResponseWriter, req *http.Request) {

	vars := mux.Vars(req)
	accoId := vars["accoId"]
	reservationId := vars["reservationId"]
	price := vars["price"]

	var stepenStudija string
	d := json.NewDecoder(req.Body)
	d.Decode(&stepenStudija)

	err := rh.repo.UpdateReservationByAcco(accoId, reservationId, price)
	if err != nil {
		rh.logger.Print("Database exception: ", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	res.WriteHeader(http.StatusCreated)
}

func (rh *reservationHandler) MiddlewareReservationForAccoDeserialization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		reservationByAcco := &ReservationByAccommodation{}
		err := reservationByAcco.FromJSON(h.Body)
		if err != nil {
			http.Error(rw, "Unable to decode json", http.StatusBadRequest)
			rh.logger.Fatal(err)
			return
		}
		ctx := context.WithValue(h.Context(), KeyProduct{}, reservationByAcco)
		h = h.WithContext(ctx)
		next.ServeHTTP(rw, h)
	})
}

func (rh *reservationHandler) MiddlewareReservationForUserDeserialization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		reservationByUser := &ReservationByUser{}
		err := reservationByUser.FromJSON(h.Body)
		if err != nil {
			http.Error(rw, "Unable to decode json", http.StatusBadRequest)
			rh.logger.Fatal(err)
			return
		}
		ctx := context.WithValue(h.Context(), KeyProduct{}, reservationByUser)
		h = h.WithContext(ctx)
		next.ServeHTTP(rw, h)
	})
}

func (rh *reservationHandler) MiddlewareRoleCheck(client *http.Client, breaker *gobreaker.CircuitBreaker) mux.MiddlewareFunc {
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
				rh.logger.Println(err)
				sendErrorWithMessage(w, "Service is not working", http.StatusInternalServerError)
				return
			}

			resp := cbResp.(*http.Response)
			resBody, err := io.ReadAll(resp.Body)
			rh.logger.Println("Host Id:", string(resBody))
			if resp.StatusCode != http.StatusOK {
				rh.logger.Println("Error in auth response " + strconv.Itoa(resp.StatusCode))
				rh.logger.Println("status " + resp.Status)
				w.WriteHeader(resp.StatusCode)
				return
			}
			rh.logger.Println(resp)

			hostID := string(resBody)

			var requestBody map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				rh.logger.Println("Error decoding request body:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if startDateStr, ok := requestBody["startDate"].(string); ok { //parsiranje datuma
				startDate, err := time.Parse("2006-01-02", startDateStr)
				if err != nil {
					rh.logger.Println("Error parsing startDate:", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				requestBody["startDate"] = startDate
			}

			if endDateStr, ok := requestBody["endDate"].(string); ok { //
				endDate, err := time.Parse("2006-01-02", endDateStr)
				if err != nil {
					rh.logger.Println("Error parsing endDate:", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				requestBody["endDate"] = endDate
			}

			hostID = strings.Trim(hostID, `"`)
			requestBody["userId"] = hostID

			modifiedJSON, err := json.Marshal(requestBody)
			if err != nil {
				rh.logger.Println("Error marshaling modified JSON:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			newReq, err := http.NewRequestWithContext(ctx, r.Method, r.URL.String(), bytes.NewBuffer(modifiedJSON))
			if err != nil {
				rh.logger.Println("Error creating new request:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			newReq.Header = r.Header

			newReq.Header.Set("Content-Type", "application/json")
			rh.logger.Println(newReq.Body)

			next.ServeHTTP(w, newReq)
		})
	}
}

func (rh *reservationHandler) MiddlewareRoleCheck1(client *http.Client, breaker *gobreaker.CircuitBreaker) mux.MiddlewareFunc {
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
				rh.logger.Println(err)
				sendErrorWithMessage(w, "Service is not working", http.StatusInternalServerError)
				return
			}

			resp := cbResp.(*http.Response)
			resBody, err := io.ReadAll(resp.Body)
			rh.logger.Println("Host Id:", string(resBody))
			if resp.StatusCode != http.StatusOK {
				rh.logger.Println("Error in auth response " + strconv.Itoa(resp.StatusCode))
				rh.logger.Println("status " + resp.Status)
				w.WriteHeader(resp.StatusCode)
				return
			}
			rh.logger.Println(resp)

			next.ServeHTTP(w, r)
		})
	}
}

func (rh *reservationHandler) MiddlewareRoleCheck0(client *http.Client, breaker *gobreaker.CircuitBreaker) mux.MiddlewareFunc {
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
				rh.logger.Println(err)
				sendErrorWithMessage(w, "Service is not working", http.StatusInternalServerError)
				return
			}

			resp := cbResp.(*http.Response)
			resBody, err := io.ReadAll(resp.Body)
			rh.logger.Println("Host Id:", string(resBody))
			if resp.StatusCode != http.StatusOK {
				rh.logger.Println("Error in auth response " + strconv.Itoa(resp.StatusCode))
				rh.logger.Println("status " + resp.Status)
				w.WriteHeader(resp.StatusCode)
				return
			}

			userID := string(resBody)

			requestBody := map[string]interface{}{
				"userId": userID,
			}

			userID = strings.Trim(userID, `"`)
			requestBody["userId"] = userID

			modifiedJSON, err := json.Marshal(requestBody)
			if err != nil {
				rh.logger.Println("Error marshaling modified JSON:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			newReq, err := http.NewRequestWithContext(ctx, r.Method, r.URL.String(), bytes.NewBuffer(modifiedJSON))
			if err != nil {
				rh.logger.Println("Error creating new request:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			newReq.Header = r.Header

			newReq.Header.Set("Content-Type", "application/json")

			next.ServeHTTP(w, newReq)
		})
	}
}

func (rh *reservationHandler) MiddlewareContentTypeSet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		rh.logger.Println("Method [", h.Method, "] - Hit path :", h.URL.Path)

		rw.Header().Add("Content-Type", "application/json")

		next.ServeHTTP(rw, h)
	})
}

func decodeBody(r io.Reader) (*ReservationDateByDate, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var rt ReservationDateByDate
	if err := dec.Decode(&rt); err != nil {
		log.Println("Error u decode body:", err)
		return nil, err
	}
	err := ValidateReservationDateByDate(&rt)
	if err != nil {
		return nil, err
	}

	return &rt, nil
}

func decodeReservationBody(r io.Reader) (*ReservationByAccommodation, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var rt ReservationByAccommodation
	if err := dec.Decode(&rt); err != nil {
		log.Println("Error u decode body:", err)
		return nil, err
	}
	err := ValidateReservationByAccommodation(&rt)
	if err != nil {
		return nil, err
	}

	return &rt, nil
}

func decodeIdBody(r io.Reader) (*RequestId, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var rt RequestId
	if err := dec.Decode(&rt); err != nil {
		log.Println("Error u decode body:", err)
		return nil, err
	}

	return &rt, nil
}

func decodeReservationByUserBody(r io.Reader) (*ReservationByUser, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var rt ReservationByUser
	if err := dec.Decode(&rt); err != nil {
		log.Println("Error u decode body:", err)
		return nil, err
	}

	return &rt, nil
}

func sendErrorWithMessage1(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(message))
	w.WriteHeader(statusCode)
}

func sendErrorWithMessage(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errorResponse := map[string]string{"message": message}
	json.NewEncoder(w).Encode(errorResponse)
}

func (nh *reservationHandler) ExtractTraceInfoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
