package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gorillaHandlers "github.com/gorilla/handlers"

	//jaeger
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func main() {

	config := loadConfig()
	//
	//authClient := &http.Client{
	//	Transport: &http.Transport{
	//		MaxIdleConns:        10,
	//		MaxIdleConnsPerHost: 10,
	//		MaxConnsPerHost:     10,
	//	},
	//}
	//
	//authBreaker := gobreaker.NewCircuitBreaker(
	//	gobreaker.Settings{
	//		Name:        "auth",
	//		MaxRequests: 1,
	//		Timeout:     10 * time.Second,
	//		Interval:    0,
	//	})

	tracerProvider, err := NewTracerProvider(config["jaeger"])
	if err != nil {
		log.Fatal("JaegerTraceProvider failed to Initialize", err)
	}
	tracer := tracerProvider.Tracer("notification-service")

	//JAEGER
	//ctx := context.Background()
	//exp, err := newExporter(config["jaeger"])
	//if err != nil {
	//	log.Fatalf("failed to initialize exporter: %v", err)
	//}
	//tp := newTraceProvider(exp)
	//defer func() { _ = tp.Shutdown(ctx) }()
	//otel.SetTracerProvider(tp)
	//// Finally, set the tracer that can be used for this package.
	//tracer := tp.Tracer("notification-service")
	//otel.SetTextMapPropagator(propagation.TraceContext{})
	//----------

	timeoutContext, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	logger := log.New(os.Stdout, "[notification-api] ", log.LstdFlags)
	storeLogger := log.New(os.Stdout, "[notification-store] ", log.LstdFlags)
	//pub := InitPubSub()
	store, err := New(timeoutContext, storeLogger, tracer)
	if err != nil {
		logger.Fatal(err)
	}
	defer store.Disconnect(timeoutContext)

	store.Ping()

	router := mux.NewRouter()
	//router.StrictSlash(true)
	cors := gorillaHandlers.CORS(gorillaHandlers.AllowedOrigins([]string{"*"}))

	service := NewNotificationHandler(logger, store, tracer)

	router.Use(service.MiddlewareContentTypeSet)
	router.Use(service.ExtractTraceInfoMiddleware)

	postRouter := router.Methods(http.MethodPost).Subrouter()
	postRouter.HandleFunc("/api/notifications/create", service.createNotification)
	//postRouter.Use(service.MiddlewareRoleCheck(authClient, authBreaker))
	//postRouter.Use(service.MiddlewareAccommodationDeserialization)

	router.HandleFunc("/api/notifications/{id}", service.GetAllNotificationsByUserId).Methods("GET")

	server := http.Server{
		Addr:         ":" + config["port"],
		Handler:      cors(router),
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	log.Println(config["jaeger"])
	logger.Println(config["jaeger"])
	logger.Println("Server listening on port", config["port"])
	//Distribute all the connections to goroutines
	go func() {
		// err := server.ListenAndServe()
		err := server.ListenAndServeTLS("/cert/notification-service.crt", "/cert/notification-service.key")
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

func loadConfig() map[string]string {
	config := make(map[string]string)
	config["host"] = os.Getenv("HOST")
	config["port"] = os.Getenv("PORT")
	config["address"] = fmt.Sprintf(":%s", os.Getenv("PORT"))
	config["conn_reservation_service_address"] = fmt.Sprintf("https://%s:%s", os.Getenv("RESERVATION_SERVICE_HOST"), os.Getenv("RESERVATION_SERVICE_PORT"))
	config["jaeger"] = os.Getenv("JAEGER_ADDRESS")
	return config
}

func newExporter(address string) (*jaeger.Exporter, error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(address)))
	if err != nil {
		return nil, err
	}
	return exp, nil
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
			semconv.ServiceNameKey.String("notification-service"),
			semconv.DeploymentEnvironmentKey.String("development"),
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil
}

func newTraceProvider(exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	// Ensure default SDK resources and the required service name are set.
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("notification-service"),
		),
	)

	if err != nil {
		panic(err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
}
