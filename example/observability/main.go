package main

import (
	"fmt"
	"time"

	"github.com/ocuris/flux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Declare our metrics
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "flux_requests_total",
			Help: "Count of all HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "flux_request_duration_seconds",
			Help:    "Duration of all HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	// Register metrics with Prometheus
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpDuration)
}

// PrometheusMiddleware bridge between Flux and Prometheus
func PrometheusMiddleware(next flux.HandlerFunc) flux.HandlerFunc {
	return func(c *flux.Context) error {
		start := time.Now()

		// Run the next handler
		err := next(c)

		// Record metrics after request finishes
		duration := time.Since(start).Seconds()
		status := "200" // Default for demo

		// Record labels
		httpRequestsTotal.WithLabelValues(c.Request.Method, c.Request.URL.Path, status).Inc()
		httpDuration.WithLabelValues(c.Request.Method, c.Request.URL.Path).Observe(duration)

		return err
	}
}

func main() {
	app := flux.New(flux.Config{
		Title:       "Flux Observability Demo",
		Description: "Monitoring example",
		Debug:       true,
	})

	// 1. Add our metrics middleware
	app.Use(PrometheusMiddleware)

	// 2. Add a standard Prometheus scrape endpoint
	// promhttp.Handler() returns a standard http.Handler, so we can mount it directly.
	app.GET("/metrics", func(c *flux.Context) error {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
		return nil
	})

	app.GET("/hello", func(c *flux.Context) error {
		return c.String(200, "Observability is enabled!")
	})

	fmt.Println("📊 Metrics available at http://localhost:8080/metrics")
	app.Start(":8080")
}
