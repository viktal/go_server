package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

var totalRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Number of get requests",
	},
	[]string{"path"})

var responseStatus = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "response_status",
		Help: "Status of HTTP response",
	},
	[]string{"status"})

var httpDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "http_response_time_seconds",
		Help: "Duration of HTTP requests",
	},
	[]string{"path"})

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()

		timer := prometheus.NewTimer(httpDuration.WithLabelValues(path))
		rw := NewResponseWriter(w)
		next.ServeHTTP(rw, r)

		statusCode := rw.statusCode

		responseStatus.WithLabelValues(strconv.Itoa(statusCode)).Inc()
		totalRequests.WithLabelValues(path).Inc()
		timer.ObserveDuration()
	})
}

func init() {
	err := prometheus.Register(totalRequests)
	if err != nil {
		return
	}
	err = prometheus.Register(responseStatus)
	if err != nil {
		return
	}
	err = prometheus.Register(httpDuration)
	if err != nil {
		return
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Duration(rand.Float64() * 2) * time.Second)
	randomInt := 5 + rand.Intn(10)
	fib1 := 1
	fib2 := 1
	for i := 2; i < randomInt; i ++ {
		fibSum := fib1 + fib2
		fib1 = fib2
		fib2 = fibSum
	}

	fmt.Fprintf(w, "Hello World")
}

func main() {
	router := mux.NewRouter()
	router.Use(prometheusMiddleware)

	router.HandleFunc("/", handler)
	router.Path("/prometheus").Handler(promhttp.Handler())

	fmt.Println("Serving request on port 9000")
	err := http.ListenAndServe(":9000", router)
	if err != nil {
		log.Fatal(err)
	}
}

