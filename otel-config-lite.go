package main

import (
	// Install required packages
	// go get "go.opentelemetry.io/contrib/exporters/autoexport" "go.opentelemetry.io/contrib/bridges/otelslog"
	"os"
	"time"
	"context"
	"log/slog"
	"net"
	"net/url"
	"strings"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Set what endpoint you're going to send your telemetry data to
const OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:4318"

// true: also log to stdout. false: don't log to stdout. (will export to otel regardless if endpoint works)
const LOGTOSTDOUT = true

func init() {
	// If collector isn't running, skip setup to save the user unnecessary export error warnings
	if !isOtelCollectorAvailable() {
		return
	}
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", OTEL_EXPORTER_OTLP_ENDPOINT)

	ctx := context.Background()
	const exportTimeout = 0 * time.Second  // don't retry so we don't get stuck if there's no ingest server
	/////////////////////////////
	// Configure traces export //
	/////////////////////////////
	traceExporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		panic(err)
	}
	spanProcessor := sdktrace.NewBatchSpanProcessor(traceExporter, sdktrace.WithBatchTimeout(exportTimeout))
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanProcessor))
	// sdktrace.WithBatcher(traceExporter)
	otel.SetTracerProvider(tracerProvider)


	//////////////////////////////
	// Configure metrics export //
	//////////////////////////////
	// metricReader, err := autoexport.NewMetricReader(ctx)
	// if err != nil {
	// 	panic(err)
	// }
	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		panic(err)
	}
	metricReader := sdkmetric.NewPeriodicReader(
		metricExporter,
		sdkmetric.WithTimeout(exportTimeout),
	)
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(metricReader))
	otel.SetMeterProvider(meterProvider)

	///////////////////////////
	// Configure logs export //
	///////////////////////////
	logExporter, err := autoexport.NewLogExporter(ctx)
	if err != nil {
		panic(err)
	}

	logProcessor := sdklog.NewBatchProcessor(logExporter, sdklog.WithExportTimeout(exportTimeout))
	logProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(logProcessor),
	)
	global.SetLoggerProvider(logProvider)

	logHandlers := []slog.Handler{
		// Handler to send logs to otel endpoint
		otelslog.NewHandler("otel"),
	}
	if LOGTOSTDOUT {
		// Handler to send logs to stdout. Set to the most permissive level (LevelDebug), don't change this
		logHandlers = append(logHandlers, slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	logger := slog.New(NewMultiHandler(logHandlers...))
	slog.SetDefault(logger)

	/////////////////////////////////////////////
	// Examples of how to instrument your code //
	/////////////////////////////////////////////
	// Create Logs. multiple ways:
	// Any logs created from the "log" or "slog" package are automatically exported
	// log.Println("Otel configured")
	// slog.Info("Otel configured")

	// Create Metrics
	// import "go.opentelemetry.io/otel/metric"
	// import "go.opentelemetry.io/otel/attribute"
	// meter := otel.Meter("test_meter")  // this name doesn't really matter
	// testCnt, err := meter.Int64Counter("a_test_counter")
	// if err != nil {
	// 	panic(err)
	// }
	// metricAttr := attribute.Int("test_attr", 7)  // optionall add an attribute
	// testCnt.Add(ctx, 1, metric.WithAttributes(metricAttr))  // Increment the counter

	// Create Traces
	// tracer := otel.Tracer("test_tracer")
	// ctx, span := tracer.Start(ctx, "test_span")
	// (operation should occur completely between span creation and end)
	// defer span.End()


	// Ideally we'd call these in main(), but passing it to main brakes the isolation I want for this file
	// Calling it here in init() breaks things (logs) since it's called before the rest of the program
	// Not calling it at all seems to be fine.
	// defer tracerProvider.Shutdown(ctx)
	// defer logProvider.Shutdown(ctx)
	// defer meterProvider.Shutdown(ctx)
}


func isOtelCollectorAvailable() bool {
	address := tryStripHTTPPrefix(OTEL_EXPORTER_OTLP_ENDPOINT)
	conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Remove the "http(s)://" part of the address if it's there
func tryStripHTTPPrefix(address string) string {
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		u, err := url.Parse(address)
		if err == nil {
			return u.Host  // stripped prefix
		}
	}
	return address
}

// I wanted to avoid an additionaly package installation so I manually implemented this for simplicity
// This is simply so that we can forward logs both to stdout and export them through otel
// https://pkg.go.dev/github.com/samber/slog-multi#section-readme
type MultiHandler struct {
    handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
    return &MultiHandler{handlers: handlers}
}

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
    // Return true if any handler is enabled
    for _, handler := range h.handlers {
        if handler.Enabled(ctx, level) {
            return true
        }
    }
    return false
}

func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
    for _, handler := range h.handlers {
        if err := handler.Handle(ctx, r); err != nil {
            return err
        }
    }
    return nil
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    newHandlers := make([]slog.Handler, len(h.handlers))
    for i, handler := range h.handlers {
        newHandlers[i] = handler.WithAttrs(attrs)
    }
    return NewMultiHandler(newHandlers...)
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
    newHandlers := make([]slog.Handler, len(h.handlers))
    for i, handler := range h.handlers {
        newHandlers[i] = handler.WithGroup(name)
    }
    return NewMultiHandler(newHandlers...)
}


