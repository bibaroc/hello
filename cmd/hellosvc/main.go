package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/oklog/oklog/pkg/group"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	logger := log.NewLogfmtLogger(os.Stderr)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.DefaultCaller)

	var g group.Group

	{ // hello service
		httpListener, err := net.Listen("tcp", ":8080")
		if err != nil {
			_ = logger.Log("transport", "HTTP", "during", "Listen", "err", err)
			os.Exit(1)
		}
		g.Add(func() error {
			app := http.NewServeMux()
			app.HandleFunc("/", sayHelloWrp(makeHelloSVCMetrics(), logger))

			_ = logger.Log("transport", "HTTP", "addr", ":8080")
			return http.Serve(httpListener, app)
		}, func(error) {
			httpListener.Close()
		})
	}

	{ // metrics
		httpListener, err := net.Listen("tcp", ":9100")
		if err != nil {
			_ = logger.Log("transport", "HTTP", "during", "Listen", "err", err)
			os.Exit(1)
		}
		g.Add(func() error {
			app := http.NewServeMux()
			app.Handle("/metrics", promhttp.Handler())

			_ = logger.Log("transport", "HTTP", "addr", ":9100")
			return http.Serve(httpListener, app)
		}, func(error) {
			httpListener.Close()
		})
	}

	{ // graceful shutdown
		cancelInterrupt := make(chan struct{})
		g.Add(func() error {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			select {
			case sig := <-c:
				return fmt.Errorf("received signal %s", sig)
			case <-cancelInterrupt:
				return nil
			}
		}, func(error) {
			close(cancelInterrupt)
		})
	}

	_ = logger.Log("exit", g.Run())
}

type helloSVCMetrics struct {
	requestCount   metrics.Counter
	requestSize    metrics.Counter
	requestLatency metrics.Histogram
}

func makeHelloSVCMetrics() helloSVCMetrics {
	fieldKeys := []string{"method", "error"}

	return helloSVCMetrics{
		requestCount: kitprometheus.NewCounterFrom(prometheus.CounterOpts{
			Namespace: "dyslav",
			Subsystem: "hello_svc",
			Name:      "request_count",
			Help:      "Total number of requests received.",
		}, fieldKeys),
		requestSize: kitprometheus.NewCounterFrom(prometheus.CounterOpts{
			Namespace: "dyslav",
			Subsystem: "hello_svc",
			Name:      "request_size",
			Help:      "Size of requests recieved",
		}, fieldKeys),
		requestLatency: kitprometheus.NewHistogramFrom(prometheus.HistogramOpts{
			Namespace: "dyslav",
			Subsystem: "hello_svc",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, fieldKeys),
	}
}

func sayHelloWrp(m helloSVCMetrics, l log.Logger) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		var err error

		defer func(begin time.Time) {
			lvs := []string{"method", "sayHello", "error", fmt.Sprint(err != nil)}
			m.requestCount.With(lvs...).Add(1)
			m.requestSize.With(lvs...).Add(float64(r.ContentLength))
			m.requestLatency.With(lvs...).Observe(float64(time.Since(begin).Microseconds()))
		}(time.Now())

		defer func(begin time.Time) {
			_ = l.Log(
				"method", "sayHello",
				"request_size", r.ContentLength,
				"dt", time.Since(begin).String(),
				"err", err,
			)
		}(time.Now())

		err = sayHello(rw, r)
	}
}

func sayHello(rw http.ResponseWriter, r *http.Request) error {
	reqDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		return err
	}

	rw.Header().Set("Content-Type", "text/plain")
	rw.WriteHeader(http.StatusOK)

	_, err = rw.Write(reqDump)

	return err
}
