// Package httpserver реализует служебный HTTP-сервер процесса.
package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/ZheglY/Interview-Material/internal/availability"
	"github.com/ZheglY/Interview-Material/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Server управляет служебным HTTP endpoint без бизнес-методов авторизации.
type Server struct {
	http *http.Server
}

// New создаёт HTTP-сервер с отдельным mux и обязательными таймаутами.
func New(
	cfg config.HTTPConfig,
	probe *availability.Probe,
	registry *prometheus.Registry,
	logger *zap.Logger,
) *Server {
	handler := NewHandler(probe, registry)

	return &Server{
		http: &http.Server{
			Addr:              cfg.Address,
			Handler:           handler,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			MaxHeaderBytes:    1 << 20, // 1 MiB — верхняя граница HTTP headers.
			ErrorLog:          zap.NewStdLog(logger.Named("http")),
		},
	}
}

// NewHandler создаёт маршруты отдельно от network server для быстрых unit-тестов.
func NewHandler(probe *availability.Probe, registry *prometheus.Registry) http.Handler {
	mux := http.NewServeMux()

	// Liveness отвечает, пока сам процесс способен обрабатывать HTTP-запросы.
	mux.Handle("GET /live", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeStatus(writer, http.StatusOK, "alive")
	}))

	// Readiness зависит от запуска и остановки внутренних transport-компонентов.
	mux.Handle("GET /ready", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !probe.IsReady() {
			writeStatus(writer, http.StatusServiceUnavailable, "not_ready")
			return
		}
		writeStatus(writer, http.StatusOK, "ready")
	}))

	// promhttp сериализует содержимое изолированного Prometheus registry.
	mux.Handle("GET /metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	return mux
}

// Serve принимает HTTP-запросы через заранее открытый listener.
func (server *Server) Serve(listener net.Listener) error {
	err := server.http.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown перестаёт принимать новые соединения и ждёт активные HTTP-запросы.
func (server *Server) Shutdown(ctx context.Context) error {
	return server.http.Shutdown(ctx)
}

// writeStatus отправляет маленький JSON-ответ, который нельзя кэшировать.
func writeStatus(writer http.ResponseWriter, statusCode int, state string) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(statusCode)

	// Ошибка записи уже означает разорванное клиентом соединение; повторно ответить нельзя.
	_ = json.NewEncoder(writer).Encode(map[string]string{"status": state})
}

// Общая логика файла:
//  1. /live проверяет, что процесс работает.
//  2. /ready сообщает, можно ли направлять процессу новый трафик.
//  3. /metrics отдаёт технические метрики Prometheus.
//  4. Бизнес HTTP API и Swagger появятся в отдельном gateway, а не здесь.


