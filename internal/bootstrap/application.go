package bootstrap

import (
	"context"
	"fmt"
	"net"

	"github.com/ZheglY/Interview-Material/internal/availability"
	"github.com/ZheglY/Interview-Material/internal/config"
	"github.com/ZheglY/Interview-Material/internal/observability"
	grpcserver "github.com/ZheglY/Interview-Material/internal/transport/grpc"
	httpserver "github.com/ZheglY/Interview-Material/internal/transport/grpc/http"
	"go.uber.org/zap"
)

// Application владеет listeners и серверами одного запущенного процесса.
type Application struct {
	config config.Config
	logger *zap.Logger
	probe *availability.Probe
	grpcServer *grpcserver.Server
	httpServer *httpserver.Server
	grpcListener net.Listener
	httpListener net.Listener
}

// serverResult сообщает, какой transport завершился и с какой ошибкой.
type serverResult struct {
	name string
	err error
}

// Создает Application и заранее занимает оба TCP-порта.
func New(cfg config.Config, logger *zap.Logger) (*Application, error) {
	probe := availability.NewProbe() // по умолчанию false
	registry := observability.NewRegistry()

	grpcServer := grpcserver.New(cfg.GRPC, logger.Named("grpc"))
	httpServer := httpserver.New(cfg.HTTP, probe, registry, logger.Named("adminHTTP"))

	// открываем адрес для gRPC и начинаем владеть портом.
	grpcListener, err := net.Listen("tcp", cfg.GRPC.Address)
	if err != nil {
		return nil, fmt.Errorf("открыть gRPC listener %q: %w", cfg.GRPC.Address, err)
	}

	// открываем адрес для HTTP и начинаем владеть портом.
	httpListener, err := net.Listen("tcp", cfg.HTTP.Address)
	if err != nil {
		return nil, fmt.Errorf("открыть HTTP listener %q: %w", cfg.HTTP.Address, err)
	}

	return &Application{
		config: cfg,
		logger: logger,
		probe: probe,
		grpcServer: grpcServer,
		httpServer: httpServer,
		grpcListener: grpcListener,
		httpListener: httpListener,
	}, nil
}

// Run запускает transport-серверы
func (application *Application) Run(ctx context.Context) error {
	//
	result := make(chan serverResult, 2)

	// запуск gRPC сервера
	go func() {
		result <- serverResult{
			name: "grpc",
			err: application.grpcServer.Serve(application.grpcListener),
		}
	}()

	// запуск HTTP сервера
	go func() {
		result <- serverResult{
			name: "http",
			err: application.httpServer.Serve(application.httpListener),
		}
	}()
	
	// Порты уже открыты, goroutine запущены — экземпляр можно включать в балансировку.
	application.grpcServer.MarkServing() // готовность указывается через специальную библиотеку
	application.probe.IsReady() // готовность указывается через структуру Probe
	
	application.logger.Info(
		"серверы запущены",
		zap.String("grpc_address", application.grpcListener.Addr().String()),
		zap.String("http_address", application.httpListener.Addr().String()),
		zap.Bool("grpc_reflection", application.config.GRPC.Reflection),
	)

	

}