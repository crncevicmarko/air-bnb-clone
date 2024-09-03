package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/vukasinc25/fst-airbnb/token"

	"github.com/gorilla/mux"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	AuthorizationPayloadKey = "authorization_payload"
	AccessTokenKey          = "accessToken"
)

// AuthMiddleware creates a Gorilla middleware for authorization
func AuthMiddleware(tokenMaker token.Maker) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Retrieve the authorization header from the request
			authorizationHeader := r.Header.Get(authorizationHeaderKey)

			if len(authorizationHeader) == 0 {
				// If authorization header is not provided, return an error
				err := errors.New("authorization header is not provided")
				writeError(w, http.StatusUnauthorized, err)
				return
			}

			// Split the authorization header into fields
			fields := strings.Fields(authorizationHeader)
			if len(fields) < 2 {
				// If the authorization header format is invalid, return an error
				err := errors.New("invalid authorization header format")
				writeError(w, http.StatusUnauthorized, err)
				return
			}

			// Extract the authorization type
			authorizationType := strings.ToLower(fields[0])
			if authorizationType != authorizationTypeBearer {
				// If the authorization type is not supported, return an error
				err := fmt.Errorf("unsupported authorization type %s", authorizationType)
				writeError(w, http.StatusUnauthorized, err)
				return
			}

			// Extract the access token
			accessToken := fields[1]
			payload, err := tokenMaker.VerifyToken(accessToken)
			if err != nil {
				// If the token verification fails, return an error
				writeError(w, http.StatusUnauthorized, err)
				return
			}

			// Store the payload in the request context
			ctx := context.WithValue(r.Context(), AuthorizationPayloadKey, payload)
			ctx = context.WithValue(ctx, AccessTokenKey, accessToken)
			r = r.WithContext(ctx)

			//xss handling
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			// Call the next handler in the chain
			next.ServeHTTP(w, r)
		})
	}
}

func AuthMiddleware1(tokenMaker token.Maker) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("req received")

			dec := json.NewDecoder(r.Body)

			var rt ReqToken
			err := dec.Decode(&rt)
			if err != nil {
				log.Println(err)
				log.Println("Request decode error")
			}

			log.Println(rt.Token)

			payload, err := tokenMaker.VerifyToken(rt.Token)
			if err != nil {
				// If the token verification fails, return an error
				log.Println("error in token verification")
				writeError(w, http.StatusUnauthorized, err)
				return
			}

			// respBytes, err := json.Marshal(payload.ID)
			// if err != nil {
			// 	log.Println("error while creating response")
			// 	w.WriteHeader(http.StatusInternalServerError)
			// 	return
			// }

			// w.Header().Add("Content-Type", "application/json")
			// w.Write(respBytes)
			// 	})
			r = r.WithContext(context.WithValue(r.Context(), AuthorizationPayloadKey, payload))
			next.ServeHTTP(w, r)
		})
	}
}

// writeError writes an error response to the client
func writeError(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
}

func SetCSPHeader(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// pomaze browseru da ucitava podatke istog porekla kao on
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		// pomaze browseru ako detektuje xss napad da ne dozvoli ucitavanje
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}
