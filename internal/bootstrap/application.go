package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

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

	var runError error

	select {
	case <-ctx.Done():
		application.logger.Info("получен сигнал завершения")
	case result := <-result:
		// Самопроизвольная остановка одного transport считается аварией процесса.
		if result.err == nil {
			runError = fmt.Errorf("%s сервер неожиданно остановился", result.name)
		} else {
			runError = fmt.Errorf("%s сервер завершился с ошибкой: %w", result.name, result.err)
		}
	}

	// Для остановки создаётся новый context: входной уже может быть отменён сигналом.
	shutdownContext, cancel := context.WithTimeout(context.Background(), application.config.ShutdownTimeout)
	defer cancel()

	if err := application.shutdown(shutdownContext); err != nil {
		if runError == nil {
			runError = err
		} else {
			runError = errors.Join(runError, err)
		}
	}

	return runError


}

// shutdown параллельно останавливает HTTP и gRPC в пределах общего timeout.
func (application *Application) shutdown(ctx context.Context) error {
	application.probe.MarkNotReady()

	var waitGroup sync.WaitGroup
	errorChannel := make(chan error, 1)

	waitGroup.Add(2)
	go func() {
		defer waitGroup.Done()
		application.grpcServer.GracefulStop(ctx)
	}()

	go func() {
		defer waitGroup.Done()
		if err := application.httpServer.Shutdown(ctx); err != nil {
			errorChannel <- fmt.Errorf("остановить HTTP-сервер: %w", err)
		}
	}()

	waitGroup.Wait()
	close(errorChannel)

	var shutdownError error
	for err := range errorChannel {
		shutdownError = errors.Join(shutdownError, err)
	}

	if shutdownError == nil {
		application.logger.Info("серверы корректно остановлены")
	}

	return shutdownError
}

//  1. New создаёт зависимости и синхронно открывает оба listener.
//  2. Run запускает gRPC и HTTP, после чего включает readiness.
//  3. Сигнал ОС или ошибка любого server запускает общий shutdown.
//  4. Новые запросы прекращаются, а активным даётся ограниченное время.