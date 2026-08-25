package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ZheglY/Interview-Material/internal/bootstrap"
	"github.com/ZheglY/Interview-Material/internal/buildinfo"
	"github.com/ZheglY/Interview-Material/internal/config"
	loggerfactory "github.com/ZheglY/Interview-Material/internal/logger"
	"go.uber.org/zap"
)

// main преобразует код завершения run в exit code операционной системы.
func main() {
	os.Exit(run())
}

// run выполняет последовательный bootstrap и возвращает контролируемый exit code.
func run() int {
	applicationConfig, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ошибка конфигурации: %v\n", err)
		return 1
	}

	logger, err := loggerfactory.New(applicationConfig.Environment, applicationConfig.Log.Level)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ошибка логгера: %v\n", err)
		return 1
	}

	// Sync сбрасывает буферы; ошибка синхронизации stdout на Windows не влияет на exit code.
	defer func() { _ = logger.Sync() } ()

		logger.Info(
		"запуск auth-сервиса",
		zap.String("environment", applicationConfig.Environment),
		zap.String("version", buildinfo.Version),
		zap.String("commit", buildinfo.Commit),
		zap.String("build_time", buildinfo.BuildTime),
	)

	// Контекст отменяется стандартным Ctrl+C или SIGTERM от container orchestrator.
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	application, err := bootstrap.New(applicationConfig, logger)
	if err != nil {
		logger.Error("не удалось собрать приложение", zap.Error(err))
		return 1
	}

	if err := application.Run(ctx); err != nil {
		logger.Error("приложение завершилось с ошибкой", zap.Error(err))
		return 1
	}

	logger.Info("auth-сервис завершён")
	return 0
}

//  1. main ничего не знает о gRPC-методах и только запускает run.
//  2. run загружает config, создаёт zap и регистрирует системные сигналы.
//  3. bootstrap.Application владеет transport-компонентами и их shutdown.
//  4. Любая ошибка запуска преобразуется в ненулевой process exit code.

