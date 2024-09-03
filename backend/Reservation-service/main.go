package main

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	// "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	lumberjack "github.com/natefinch/lumberjack"
	log "github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
	saga "github.com/vukasinc25/fst-airbnb/utility/saga/messaging"
	nats "github.com/vukasinc25/fst-airbnb/utility/saga/messaging/nats"

	// handlers "github.com/vukasinc25/fst-airbnb/handlers"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const (
	QueueGroup = "reservation_service"
)

func main() {

	logger := log.New()
	config := loadConfig()
	// Set up log rotation with Lumberjack
	lumberjackLogger := &lumberjack.Logger{
		Filename:   "/res/file.log",
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

	//config := loadConfig()

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

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8000"
	}

	//TRACING
	tracerProvider, err := NewTracerProvider(config["jaeger"])
	if err != nil {
		log.Fatal("JaegerTraceProvider failed to Initialize", err)
	}
	tracer := tracerProvider.Tracer("Reservation-service")
	//

	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// logger := log.New(os.Stdout, "[reservation-api] ", log.LstdFlags)
	// storeLogger := log.New(os.Stdout, "[reservation-store] ", log.LstdFlags)

	store, err := New(logger, tracer)
	if err != nil {
		logger.Fatal(err)
	}
	defer store.CloseSession()
	store.CreateTables()

	commandSubscriber := initSubscriber(os.Getenv("CREATE_ACCOMMODATION_COMMAND_SUBJECT"), QueueGroup) // commandSubscriber
	replyPublisher := initPublisher(os.Getenv("CREATE_ACCOMMODATION_REPLY_SUBJECT"))                   // replyPublisher
	handel := initCreateOrderHandler(store, replyPublisher, commandSubscriber)                         // commandHandle

	log.Println("Reservation handel method:", handel)
	reservationHandler := NewReservationHandler(logger, store, tracer)
	router := mux.NewRouter()
	router.Use(reservationHandler.ExtractTraceInfoMiddleware)

	getReservationIds := router.Methods(http.MethodGet).Subrouter()
	getReservationIds.HandleFunc("/api/reservations/", reservationHandler.GetAllReservationIds)
	getReservationIds.Use(reservationHandler.ExtractTraceInfoMiddleware)

	getReservationsByAcco := router.Methods(http.MethodGet).Subrouter()
	getReservationsByAcco.HandleFunc("/api/reservations/by_acco/{id}", reservationHandler.GetAllReservationsByAccommodationId)
	getReservationsByAcco.Use(reservationHandler.MiddlewareRoleCheck1(authClient, authBreaker))
	getReservationsByAcco.Use(reservationHandler.ExtractTraceInfoMiddleware)

	getReservationsByUser := router.Methods(http.MethodGet).Subrouter()
	getReservationsByUser.HandleFunc("/api/reservations/by_user", reservationHandler.GetAllReservationsByUserId)
	getReservationsByUser.Use(reservationHandler.MiddlewareRoleCheck0(authClient, authBreaker))
	getReservationsByUser.Use(reservationHandler.ExtractTraceInfoMiddleware)

	postReservationForAcco := router.Methods(http.MethodPost).Subrouter()
	postReservationForAcco.HandleFunc("/api/reservations/for_user", reservationHandler.CreateReservationForUser)
	postReservationForAcco.Use(reservationHandler.MiddlewareRoleCheck(authClient, authBreaker))
	postReservationForAcco.Use(reservationHandler.ExtractTraceInfoMiddleware)

	patchReservationForAcco := router.Methods(http.MethodPatch).Subrouter()
	patchReservationForAcco.HandleFunc("/api/reservations/for_user", reservationHandler.UpdateReservationByUser)
	patchReservationForAcco.Use(reservationHandler.MiddlewareRoleCheck(authClient, authBreaker))
	patchReservationForAcco.Use(reservationHandler.ExtractTraceInfoMiddleware)

	postReservationForUser := router.Methods(http.MethodPost).Subrouter()
	postReservationForUser.HandleFunc("/api/reservations/for_acco", reservationHandler.CreateReservationForAcco)
	postReservationForUser.Use(reservationHandler.MiddlewareRoleCheck(authClient, authBreaker))
	postReservationForUser.Use(reservationHandler.ExtractTraceInfoMiddleware)
	// postReservationDateByAccomodation := router.Methods(http.MethodPost).Subrouter()
	// postReservationDateByAccomodation.HandleFunc("/api/reservations/date_for_acoo", reservationHandler.CreateReservationDateForAccommodation)
	// postReservationDateByAccomodation.Use(reservationHandler.MiddlewareRoleCheck1(authClient, authBreaker))

	getReservationDatesByAccomodationId := router.Methods(http.MethodGet).Subrouter()
	getReservationDatesByAccomodationId.HandleFunc("/api/reservations/dates_by_acco_id/{id}", reservationHandler.GetReservationDatesByAccommodationId)
	getReservationDatesByAccomodationId.Use(reservationHandler.MiddlewareRoleCheck1(authClient, authBreaker))
	getReservationDatesByAccomodationId.Use(reservationHandler.ExtractTraceInfoMiddleware)

	getReservationDatesByDate := router.Methods(http.MethodGet).Subrouter()
	getReservationDatesByDate.HandleFunc("/api/reservations/search_by_date/{startDate}/{endDate}", reservationHandler.GetAllReservationsDatesByDate)
	getReservationDatesByDate.Use(reservationHandler.ExtractTraceInfoMiddleware)

	postReservationDateByDate := router.Methods(http.MethodPost).Subrouter()
	postReservationDateByDate.HandleFunc("/api/reservations/date_for_date", reservationHandler.CreateReservationDateForDate)
	postReservationDateByDate.Use(reservationHandler.ExtractTraceInfoMiddleware)

	getReservatinsDatesByHostId := router.Methods(http.MethodGet).Subrouter()
	getReservatinsDatesByHostId.HandleFunc("/api/reservations/for_host_id/{id}", reservationHandler.GetAllReservationsDatesByHostId)
	getReservatinsDatesByHostId.Use(reservationHandler.ExtractTraceInfoMiddleware)

	getReservatinsForUserByHostId := router.Methods(http.MethodGet).Subrouter()
	getReservatinsForUserByHostId.HandleFunc("/api/reservations/by_user_for_host_id/{userId}/{hostId}", reservationHandler.GetAllReservationsForUserIdByHostId)
	getReservatinsForUserByHostId.Use(reservationHandler.ExtractTraceInfoMiddleware)

	router.HandleFunc("/api/reservations/host/{id}", reservationHandler.IsHostProminent).Methods("GET")

	cors := gorillaHandlers.CORS(gorillaHandlers.AllowedOrigins([]string{"*"}))

	server := http.Server{
		Addr:         ":" + port,
		Handler:      cors(router),
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	logger.Println("Server listening on port", port)
	//Distribute all the connections to goroutines
	go func() {
		// err := server.ListenAndServe()
		err := server.ListenAndServeTLS("/cert/reservation-service.crt", "/cert/reservation-service.key")
		if err != nil {
			logger.Fatal(err)
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

func initPublisher(subject string) saga.Publisher {
	publisher, err := nats.NewNATSPublisher(
		os.Getenv("NATS_HOST"), os.Getenv("NATS_PORT"),
		os.Getenv("NATS_USER"), os.Getenv("NATS_PASS"), subject)
	if err != nil {
		log.Fatal(err)
	}
	return publisher
}

func initSubscriber(subject, queueGroup string) saga.Subscriber {
	subscriber, err := nats.NewNATSSubscriber(
		os.Getenv("NATS_HOST"), os.Getenv("NATS_PORT"),
		os.Getenv("NATS_USER"), os.Getenv("NATS_PASS"), subject, queueGroup)
	if err != nil {
		log.Fatal(err)
	}
	return subscriber
}

func initCreateOrderHandler(store *ReservationRepo, replyPublisher saga.Publisher, commandSubscriber saga.Subscriber) *CreateResrvationCommandHandler {
	something, err := NewCreateReservationCommandHandler(store, replyPublisher, commandSubscriber) // commandHandle
	if err != nil {
		log.Fatal("Ovde1: ", err)
	}
	return something
}

func loadConfig() map[string]string {
	config := make(map[string]string)
	config["host"] = os.Getenv("HOST")
	config["port"] = os.Getenv("PORT")
	config["address"] = fmt.Sprintf(":%s", os.Getenv("PORT"))
	config["jaeger"] = os.Getenv("JAEGER_ADDRESS")
	config["conn_reservation_service_address"] = fmt.Sprintf("http://%s:%s", os.Getenv("RESERVATION_SERVICE_HOST"), os.Getenv("RESERVATION_SERVICE_PORT"))
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
			semconv.ServiceNameKey.String("Reservation-service"),
			semconv.DeploymentEnvironmentKey.String("development"),
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil
}
