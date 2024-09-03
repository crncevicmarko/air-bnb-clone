package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/gorilla/mux"
	lumberjack "github.com/natefinch/lumberjack"
	log "github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
	"github.com/vukasinc25/fst-airbnb/token"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func main() {

	logger := log.New()

	// Set up log rotation with Lumberjack
	lumberjackLogger := &lumberjack.Logger{
		Filename:   "/prof/file.log",
		MaxSize:    10, // MB
		MaxBackups: 3,
		LocalTime:  true, // Use local time
	}
	logger.SetOutput(lumberjackLogger)

	// Handle log rotation gracefully on program exit
	defer func() {
		if err := lumberjackLogger.Close(); err != nil {
			log.Error("Error closing log file:", err)
		}
	}()

	// ... (rest of your code)

	// Example log statements
	logger.Info("lavor1")

	authClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			MaxConnsPerHost:     10,
		},
	}

	authBreaker := gobreaker.NewCircuitBreaker(
		gobreaker.Settings{
			Name:        "auth",
			MaxRequests: 1,
			Timeout:     10 * time.Second,
			Interval:    0,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures > 2
			},
			OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
				log.Printf("Circuit Breaker '%s' changed from '%s' to, %s'\n", name, from, to)
			},
			IsSuccessful: func(err error) bool {
				if err == nil {
					return true
				}
				errResp, ok := err.(ErrResp)
				return ok && errResp.StatusCode >= 400 && errResp.StatusCode < 500
			},
		})

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	router := mux.NewRouter()
	router.StrictSlash(true)

	config := loadConfig()

	tokenMaker, err := token.NewJWTMaker("12345678901234567890123456789012")
	if err != nil {
		logger.Fatal(err)
	}

	//TRACING
	tracerProvider, err := NewTracerProvider(config["jaeger"])
	if err != nil {
		log.Fatal("JaegerTraceProvider failed to Initialize", err)
	}
	tracer := tracerProvider.Tracer("prof-service")
	//

	//Initialize the logger we are going to use, with prefix and datetime for every log
	// logger := log.New(os.Stdout, "[product-api] ", log.LstdFlags)
	// logger := log.New()

	// lumberjackLogger := &lumberjack.Logger{
	// 	Filename:   "/cert/misc.log",
	// 	MaxSize:    10,
	// 	MaxBackups: 3,
	// 	MaxAge:     3,
	// 	LocalTime:  true,
	// }
	// logger.SetOutput(lumberjackLogger)

	// NoSQL: Initialize Product Repository store
	store, err := New(logger, config["conn_reservation_service_address"], config["conn_auth_service_address"], tracer)
	if err != nil {
		logger.Fatal(err)
	}

	service := NewUserHandler(logger, store, tracer)

	router.Use(service.ExtractTraceInfoMiddleware)
	// router.HandleFunc("/api/prof/email/{code}", service.verifyEmail).Methods("POST") // for sending verification mail
	router.HandleFunc("/api/prof/create", service.createUser).Methods("POST")
	router.HandleFunc("/api/prof/users/", service.getAllUsers).Methods("GET")

	getUserInfoByUserId := router.Methods(http.MethodGet).Subrouter()
	getUserInfoByUserId.HandleFunc("/api/prof/user", service.GetUserById)
	getUserInfoByUserId.Use(service.MiddlewareRoleCheck(authClient, authBreaker))

	router.Methods(http.MethodPatch).Subrouter()
	getAllHostGrades := router.Methods(http.MethodGet).Subrouter()
	getAllHostGrades.HandleFunc("/api/prof/hostGrades/{id}", service.GetAllHostGrades) // treba authorisation
	getAllHostGrades.Use(service.MiddlewareRoleCheck00(authClient, authBreaker, tokenMaker))

	createHostGrade := router.Methods(http.MethodPost).Subrouter()
	createHostGrade.HandleFunc("/api/prof/hostGrade", service.CreateHostGrade) // treba authorisation
	createHostGrade.Use(service.MiddlewareRoleCheck0(authClient, authBreaker, tokenMaker))

	deleteHostGrade := router.Methods(http.MethodDelete).Subrouter()
	deleteHostGrade.HandleFunc("/api/prof/hostGrade/{id}", service.DeleteHostGrade) // treba authorisation
	deleteHostGrade.Use(service.MiddlewareRoleCheck00(authClient, authBreaker, tokenMaker))

	router.HandleFunc("/api/prof/update", service.UpdateUser).Methods("PATCH")
	router.HandleFunc("/api/prof/delete/{id}", service.DeleteUser).Methods("DELETE")

	// srv := &http.Server{Addr: config["address"], Handler: router}
	server := &http.Server{
		Addr:         ":" + "8000",
		Handler:      router,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		// TLSConfig: &tls.Config{
		// 	InsecureSkipVerify: true, // samo za testiranje
		// 	MinVersion:         tls.VersionTLS12,
		// 	CipherSuites:       []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384},
		// },
	}
	go func() {
		logger.Info("lavor4")
		// err := server.ListenAndServe()
		err := server.ListenAndServeTLS("/cert/prof-service.crt", "/cert/prof-service.key")
		if err != nil {
			logger.Println("Error starting server", err)
			// logMessage(fmt.Sprintf("Error starting server: %s", err), logrus.ErrorLevel)
		}
	}()

	<-quit

	// logMessage("Service shutting down...", logrus.InfoLevel)
	logger.Println("Service shutting down...")

	// gracefully stop server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		// logMessage(fmt.Sprintf("Error shutting down server: %s", err), logrus.ErrorLevel)
		logger.Println("Error shutting down server", err)
	}
	// logMessage("Server stopped", logrus.InfoLevel)
	logger.Println("Server stopped")

}

//	func handleErr(err error) {
//		if err != nil {
//			logger.Fatalln(err)
//		}
//	}
// func handleErr(err error) {
// 	if err != nil {
// 		// logMessage(fmt.Sprintf("Error: %s", err), logrus.ErrorLevel)
// 		logger.Println(err.Error())
// 	}
// }

func loadConfig() map[string]string {
	config := make(map[string]string)
	config["host"] = os.Getenv("HOST")
	config["port"] = os.Getenv("PORT")
	config["address"] = fmt.Sprintf(":%s", os.Getenv("PORT"))
	config["mondo_db_uri"] = os.Getenv("MONGO_DB_URI")
	config["conn_reservation_service_address"] = fmt.Sprintf("https://%s:%s", os.Getenv("RESERVATION_SERVICE_HOST"), os.Getenv("RESERVATION_SERVICE_PORT"))
	config["conn_auth_service_address"] = fmt.Sprintf("https://%s:%s", os.Getenv("AUTH_SERVICE_HOST"), os.Getenv("AUTH_SERVICE_PORT"))
	config["address"] = fmt.Sprintf(":%s", os.Getenv("PORT"))
	config["jaeger"] = os.Getenv("JAEGER_ADDRESS")
	return config
}

func NewTracerProvider(collectorEndpoint string) (*sdktrace.TracerProvider, error) {
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(collectorEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("unable to initialize exporter due: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("prof-service"),
			semconv.DeploymentEnvironmentKey.String("development"),
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil
}
