package main

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"

	// "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vukasinc25/fst-airbnb/cache"
	"github.com/vukasinc25/fst-airbnb/handlers"
	"github.com/vukasinc25/fst-airbnb/storage"
	saga "github.com/vukasinc25/fst-airbnb/utility/saga/messaging"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	lumberjack "github.com/natefinch/lumberjack"
	log "github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
	"github.com/vukasinc25/fst-airbnb/token"
	nats "github.com/vukasinc25/fst-airbnb/utility/saga/messaging/nats"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const (
	QueueGroup = "accommodation_service"
)

func main() {

	logger := log.New()

	// Set up log rotation with Lumberjack
	lumberjackLogger := &lumberjack.Logger{
		Filename:   "/acoo/file.log",
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

	config := loadConfig()

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

	//TRACING
	tracerProvider, err := NewTracerProvider(config["jaeger"])
	if err != nil {
		log.Fatal("JaegerTraceProvider failed to Initialize", err)
	}
	tracer := tracerProvider.Tracer("accommodation-service")
	//

	timeoutContext, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	// logger := log.New(os.Stdout, "[accommo-api] ", log.LstdFlags)
	// storeLogger := log.New(os.Stdout, "[accommo-store] ", log.LstdFlags)
	// storageLogger := log.New(os.Stdout, "[file-storage] ", log.LstdFlags)
	// loggerCache := log.New(os.Stdout, "[redis-cache] ", log.LstdFlags)
	//pub := InitPubSub()
	store, err := New(timeoutContext, logger, config["conn_reservation_service_address"], tracer)
	if err != nil {
		logger.Fatal("Ovde5: ", err)
	}
	defer store.Disconnect(timeoutContext)

	store.Ping()

	commandSubscriber := initSubscriber(os.Getenv("CREATE_ACCOMMODATION_COMMAND_SUBJECT"), QueueGroup) // commandSubscriber
	replyPublisher := initPublisher(os.Getenv("CREATE_ACCOMMODATION_REPLY_SUBJECT"))                   // replyPublisher
	handel := initCreateAccommodationHandler(store, replyPublisher, commandSubscriber)
	log.Println("Accommodation Handle method:", handel)

	commandPublisher := initPublisher(os.Getenv("CREATE_ACCOMMODATION_COMMAND_SUBJECT"))
	replySubscriber := initSubscriber(os.Getenv("CREATE_ACCOMMODATION_REPLY_SUBJECT"), QueueGroup)
	orchestrator := initCreateAccommodationOrchestrator(commandPublisher, replySubscriber)

	logger.Println("AccommodationOrcestrator: ", orchestrator)

	// NoSQL: Initialize File Storage store
	// imageStore, err := storage.New(storageLogger)
	imageStore, err := storage.New(logger)
	if err != nil {
		logger.Fatal("Ovde6: ", err)
	}

	// Close connection to HDFS on shutdown
	defer func() {
		if err := imageStore.Close(); err != nil {
			log.Println("Error closing image store:", err)
		}
	}()

	// Create directory tree on HDFS
	_ = imageStore.CreateDirectories()

	// prCache := cache.New(loggerCache)
	prCache := cache.New(logger)
	// Test connection
	prCache.Ping()

	//Initialize the handler and inject said logger
	storageHandler := handlers.NewStorageHandler(logger, imageStore, prCache)

	router := mux.NewRouter()
	//router.StrictSlash(true)
	cors := gorillaHandlers.CORS(gorillaHandlers.AllowedOrigins([]string{"*"}))

	service := NewAccoHandler(logger, store, storageHandler, orchestrator, tracer)

	tokenMaker, err := token.NewJWTMaker("12345678901234567890123456789012")
	if err != nil {
		logger.Fatal(err)
	}

	router.Use(service.MiddlewareContentTypeSet)
	router.Use(service.ExtractTraceInfoMiddleware)

	postRouter := router.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/api/accommodations/create", service.createAccommodation)
	postRouter.Use(service.MiddlewareRoleCheck(authClient, authBreaker, tokenMaker))
	postRouter.Use(service.MiddlewareAccommodationDeserialization)
	postRouter.Use(service.ExtractTraceInfoMiddleware)

	router.HandleFunc("/api/accommodations/", service.getAllAccommodations).Methods("GET")
	router.HandleFunc("/api/accommodations/{id}", service.GetAccommodationById).Methods("GET")
	router.HandleFunc("/api/accommodations/myAccommodations/{username}", service.GetAllAccommodationsByUsername).Methods("GET")
	router.HandleFunc("/api/accommodations/search_by_location/{locations}", service.GetAllAccommodationsByLocation).Methods("GET")
	router.HandleFunc("/api/accommodations/search_by_noGuests/{noGuests}", service.GetAllAccommodationsByNoGuests).Methods("GET")
	router.HandleFunc("/api/accommodations/search_by_date/{startDate}/{endDate}", service.GetAllAccommodationsByDate).Methods("GET")
	router.HandleFunc("/api/accommodations/get_all_acco_by_id/{id}", service.GetAllAccommodationsById).Methods("GET")
	router.HandleFunc("/api/accommodations/delete/{username}", service.DeleteAccommodation).Methods("DELETE")
	createAccommodationGrade := router.Methods(http.MethodPost).Subrouter()
	createAccommodationGrade.HandleFunc("/api/accommodations/accommodationGrade", service.GradeAccommodation) // treba authorisation
	createAccommodationGrade.Use(service.MiddlewareRoleCheck00(authClient, authBreaker, tokenMaker))
	// router.HandleFunc("/api/accommodations/accommodationGrade", service.GradeAccommodation).Methods("POST")
	getAllAccommodationGrades := router.Methods(http.MethodGet).Subrouter()
	getAllAccommodationGrades.HandleFunc("/api/accommodations/accommodationGrades/{id}", service.GetAllAccommodationGrades)
	getAllAccommodationGrades.Use(service.MiddlewareRoleCheck(authClient, authBreaker, tokenMaker))
	deleteAccommodationGrade := router.Methods(http.MethodDelete).Subrouter()
	deleteAccommodationGrade.HandleFunc("/api/accommodations/deleteAccommodationGrade/{id}", service.DeleteAccommodationGrade)
	deleteAccommodationGrade.Use(service.MiddlewareRoleCheck00(authClient, authBreaker, tokenMaker))

	router.HandleFunc("/api/accommodations/recommendations", service.GetAllRecommended).Methods("POST")

	router.HandleFunc("/api/accommodations/copy", storageHandler.CopyFileToStorage).Methods("POST")

	router.HandleFunc("/api/accommodations/write", storageHandler.WriteFileToStorage).Methods("POST")

	getAccommodationImage := router.Methods(http.MethodGet).Subrouter()
	getAccommodationImage.HandleFunc("/api/accommodations/read/{fileName}", storageHandler.ReadFileFromStorage)
	getAccommodationImage.Use(storageHandler.MiddlewareCacheHit)

	server := http.Server{
		Addr:         ":" + config["port"],
		Handler:      cors(router),
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	logger.Println("Server listening on port", config["port"])
	//Distribute all the connections to goroutines
	go func() {
		// err := server.ListenAndServe()
		err := server.ListenAndServeTLS("/cert/accommodation-service.crt", "/cert/accommodation-service.key")
		if err != nil {
			logger.Fatal("Ovde7: ", err)
		}
	}()

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGINT)
	signal.Notify(sigCh, syscall.SIGKILL)

	sig := <-sigCh
	logger.Println("Received terminate, graceful shutdown", sig)
	timeoutContext, _ = context.WithTimeout(context.Background(), 30*time.Second)

	//Try to shutdown gracefully
	if server.Shutdown(timeoutContext) != nil {
		logger.Fatal("Cannot gracefully shutdown...")
	}
	logger.Println("Server stopped")

}

func loadConfig() map[string]string {
	config := make(map[string]string)
	config["host"] = os.Getenv("HOST")
	config["port"] = os.Getenv("PORT")
	config["address"] = fmt.Sprintf(":%s", os.Getenv("PORT"))
	config["jaeger"] = os.Getenv("JAEGER_ADDRESS")
	config["conn_reservation_service_address"] = fmt.Sprintf("https://%s:%s", os.Getenv("RESERVATION_SERVICE_HOST"), os.Getenv("RESERVATION_SERVICE_PORT"))
	return config
}

func initPublisher(subject string) saga.Publisher {
	publisher, err := nats.NewNATSPublisher(
		os.Getenv("NATS_HOST"), os.Getenv("NATS_PORT"),
		os.Getenv("NATS_USER"), os.Getenv("NATS_PASS"), subject)
	if err != nil {
		log.Fatal("Ovde1: ", err)
	}
	return publisher
}

func initSubscriber(subject, queueGroup string) saga.Subscriber {
	subscriber, err := nats.NewNATSSubscriber(
		os.Getenv("NATS_HOST"), os.Getenv("NATS_PORT"),
		os.Getenv("NATS_USER"), os.Getenv("NATS_PASS"), subject, queueGroup)
	if err != nil {
		log.Fatal("Ovde2: ", err)
	}
	return subscriber
}

func initCreateAccommodationHandler(store *AccoRepo, replyPublisher saga.Publisher, commandSubscriber saga.Subscriber) *CreateAccomodationCommandHandler {
	something, err := NewCreateAccommodationCommandHandler(store, replyPublisher, commandSubscriber) // commandHandle
	if err != nil {
		log.Fatal("Ovde3: ", err)
	}

	return something
}

func initCreateAccommodationOrchestrator(publisher saga.Publisher, subscriber saga.Subscriber) *CreateAccommodationOrchestrator {
	log.Println("Publisher: ", publisher)
	log.Println("Subscriber: ", subscriber)
	orchestrator, err := NewCreateAccommodationOrchestrator(publisher, subscriber)
	if err != nil {
		log.Fatal("Ovde4: ", err)
	}
	return orchestrator
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
			semconv.ServiceNameKey.String("accommodation-service"),
			semconv.DeploymentEnvironmentKey.String("development"),
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil
}
