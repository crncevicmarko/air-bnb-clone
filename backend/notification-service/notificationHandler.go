package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/thanhpk/randstr"
	"github.com/vukasinc25/fst-airbnb/mail"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sony/gobreaker"

	"github.com/gorilla/mux"
)

type KeyProduct struct{}

type NotificationHandler struct {
	logger *log.Logger
	db     *NotificationRepo
	tracer trace.Tracer
}

func NewNotificationHandler(l *log.Logger, r *NotificationRepo, t trace.Tracer) *NotificationHandler {

	return &NotificationHandler{l, r, t}
}

func (nh *NotificationHandler) createNotification(rw http.ResponseWriter, req *http.Request) {
	ctx, span := nh.tracer.Start(req.Context(), "NotificationHandler.createNotification") //tracer
	defer span.End()                                                                      //tracer

	notification, err := decodeBody(req.Body)
	notification.Date = time.Now()
	//notification := req.Context().Value(KeyProduct{}).(*Notification)
	err = nh.db.Insert(ctx, notification)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
	}
	rw.WriteHeader(http.StatusCreated)

	if err == nil {

		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header)) //tracer

		log.Println("Usli u slanje notf maila")
		content := `
				<h1>AirBnb New notification</h1>
				<h2>You just received a new notification</h2>
				<h3>Login to see it</h3>`
		subject := "AirBnB New Notification"
		email := "vukasincadjenovic@gmail.com"
		err := nh.sendEmail(content, subject, email)
		if err != nil {
			nh.logger.Println("Notification email was not sent")
			return
		}
	} else {
		nh.logger.Println("Error")
		return
	}
}

func (nh *NotificationHandler) GetNotificationById(w http.ResponseWriter, req *http.Request) {
	ctx, span := nh.tracer.Start(req.Context(), "NotificationHandler.createNotification") //tracer
	defer span.End()

	vars := mux.Vars(req)
	id := vars["id"]

	notification, err := nh.db.GetById(id, ctx)
	if err != nil {
		nh.logger.Println(err)
	}

	if notification == nil {
		http.Error(w, "Notification with given id not found", http.StatusNotFound)
		nh.logger.Printf("Notification with id: '%s' not found", id)
		return
	}

	err = notification.ToJSON(w)
	if err != nil {
		http.Error(w, "Unable to convert to json", http.StatusInternalServerError)
		nh.logger.Fatal("Unable to convert to json :", err)
		return
	}
}

//func (nh *NotificationHandler) GetAllNotificationsByUsername(w http.ResponseWriter, req *http.Request) {
//	vars := mux.Vars(req)
//	username := vars["username"]
//
//	accommodations, err := nh.db.GetAllByUsername(username)
//	if err != nil {
//		nh.logger.Print("Database exception: ", err)
//	}
//
//	if accommodations == nil {
//		http.Error(w, "Notifications with given username not found", http.StatusNotFound)
//		nh.logger.Printf("Notifications with username: '%s' not found", username)
//		return
//	}
//
//	err = accommodations.ToJSON(w)
//	if err != nil {
//		http.Error(w, "Unable to convert to json", http.StatusInternalServerError)
//		nh.logger.Fatal("Unable to convert to json :", err)
//		return
//	}
//}

func (nh *NotificationHandler) GetAllNotificationsByUserId(w http.ResponseWriter, req *http.Request) {
	ctx, span := nh.tracer.Start(req.Context(), "NotificationHandler.createNotification") //tracer
	defer span.End()

	vars := mux.Vars(req)
	id := vars["id"]
	log.Println(id)
	accommodations, err := nh.db.GetAllByHostId(id, ctx)
	if err != nil {
		nh.logger.Print("Database exception: ", err)
	}

	if accommodations == nil {
		http.Error(w, "Notifications with given id not found", http.StatusNotFound)
		nh.logger.Printf("Notifications with id: '%s' not found", id)
		return
	}

	err = accommodations.ToJSON(w)
	if err != nil {
		http.Error(w, "Unable to convert to json", http.StatusInternalServerError)
		nh.logger.Fatal("Unable to convert to json :", err)
		return
	}
}

func (nh *NotificationHandler) DeleteNotification(res http.ResponseWriter, req *http.Request) {
	ctx, span := nh.tracer.Start(req.Context(), "NotificationHandler.createNotification") //tracer
	defer span.End()

	vars := mux.Vars(req)
	username := vars["username"]

	err := nh.db.Delete(username, ctx)
	if err != nil {
		log.Println("Error when tried to delete notification:", err)
		sendErrorWithMessage1(res, "Error when tried to delete notification", http.StatusInternalServerError)
		return
	}
	sendErrorWithMessage1(res, "User succesfully deleted", http.StatusOK)
}

func (nh *NotificationHandler) sendEmail(contentStr string, subjectStr string, email string) error {
	log.Println("SendEmail()")

	randomCode := randstr.String(20)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	log.Println(tlsConfig)

	sender := mail.NewGmailSender("Air Bnb", "mobilneaplikcijesit@gmail.com", "esrqtcomedzeapdr", tlsConfig) //postavi recoveri password
	subject := subjectStr
	content := fmt.Sprintf(contentStr, randomCode)
	to := []string{email}
	attachFiles := []string{}
	log.Println("Pre SendEmail(subject, content, to, nil, nil, attachFiles)")
	err := sender.SendEmail(subject, content, to, nil, nil, attachFiles)
	if err != nil {
		// http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println("Cant send email")
		return err
	}

	return nil

	// w.WriteHeader(http.StatusCreated)
	// message := "Poslat je mail na moblineaplikacijesit@gmail.com"
	// renderJSON(w, message)
}

func (nh *NotificationHandler) ExtractTraceInfoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (nh *NotificationHandler) MiddlewareRoleCheck00(client *http.Client, breaker *gobreaker.CircuitBreaker) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()

			reqURL := "http://auth-service:8000/api/users/auth"

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
				return client.Do(req)
			})
			if err != nil {
				nh.logger.Println(err)
				sendErrorWithMessage(w, "Service is not working", http.StatusInternalServerError)
				return
			}

			resp := cbResp.(*http.Response)
			resBody, err := io.ReadAll(resp.Body)
			nh.logger.Println("User Id:", string(resBody))
			if resp.StatusCode != http.StatusOK {
				nh.logger.Println("Error in auth response " + strconv.Itoa(resp.StatusCode))
				nh.logger.Println("status " + resp.Status)
				w.WriteHeader(resp.StatusCode)
				return
			}

			userID := string(resBody)
			ctx = context.WithValue(ctx, "userId", userID)
			ctx = context.WithValue(ctx, "accessToken", accessToken)

			newReq, err := http.NewRequestWithContext(ctx, r.Method, r.URL.String(), r.Body)
			if err != nil {
				nh.logger.Println("Error creating new request:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			newReq.Header = r.Header

			newReq.Header.Set("Content-Type", "application/json")

			log.Println("Token:", token)
			log.Println("AccessToken:", accessToken)

			next.ServeHTTP(w, newReq)
		})
	}
}

func (nh *NotificationHandler) MiddlewareNotificationDeserialization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		notification := &Notification{}
		err := notification.FromJSON(h.Body)
		if err != nil {
			http.Error(rw, "Unable to decode json", http.StatusBadRequest)
			nh.logger.Fatal(err)
			return
		}

		ctx := context.WithValue(h.Context(), KeyProduct{}, notification)
		h = h.WithContext(ctx)

		next.ServeHTTP(rw, h)
	})
}

func (nh *NotificationHandler) MiddlewareRoleCheck(client *http.Client, breaker *gobreaker.CircuitBreaker) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()
			reqURL := "http://auth-service:8000/api/users/auth"

			authorizationHeader := r.Header.Get("authorization")
			fields := strings.Fields(authorizationHeader)
			if len(fields) == 0 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			log.Println(fields)

			accessToken := fields[1]

			var token ReqToken
			token.Token = accessToken

			jsonToken, _ := json.Marshal(token)

			cbResp, err := breaker.Execute(func() (interface{}, error) {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, bytes.NewBuffer(jsonToken))
				if err != nil {
					return nil, err
				}
				return client.Do(req)
			})
			if err != nil {
				nh.logger.Println(err)
				sendErrorWithMessage(w, "Service is not working", http.StatusInternalServerError)
				return
			}

			resp := cbResp.(*http.Response)
			resBody, err := io.ReadAll(resp.Body)
			nh.logger.Println(string(resBody))
			if resp.StatusCode != http.StatusOK {
				nh.logger.Println("Error in auth response " + strconv.Itoa(resp.StatusCode))
				nh.logger.Println("status " + resp.Status)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			nh.logger.Println(resp)

			next.ServeHTTP(w, r)
		})
	}
}

func (nh *NotificationHandler) MiddlewareContentTypeSet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		nh.logger.Println("Method [", h.Method, "] - Hit path :", h.URL.Path)

		rw.Header().Add("Content-Type", "application/json")

		next.ServeHTTP(rw, h)
	})
}

func decodeBody(r io.Reader) (*Notification, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var rt Notification
	if err := dec.Decode(&rt); err != nil {
		return nil, err
	}

	err := ValidateNotification(&rt)
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
