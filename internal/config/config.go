// Package config отвечает за загрузку и проверку настроек приложения.
//
// Конфигурация читается только при запуске. Если обязательное значение имеет
// неверный формат, сервис завершает работу до открытия сетевых портов.
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// Значения по умолчанию подходят только для локальной разработки.
	defaultEnvironment     = "local"
	defaultGRPCAddress     = ":50051"
	defaultHTTPAddress     = ":8081"
	defaultShutdownTimeout = 10 * time.Second
	defaultReadHeaderTime  = 5 * time.Second
	defaultReadTime        = 10 * time.Second
	defaultWriteTime       = 15 * time.Second
	defaultIdleTime        = 60 * time.Second
	defaultMaxMessageBytes = 4 << 20 // 4 MiB защищают сервер от слишком больших сообщений.
)

// Config объединяет все настройки одного процесса auth-сервиса.
type Config struct {
	Environment     string
	GRPC            GRPCConfig
	HTTP            HTTPConfig
	Log             LogConfig
	ShutdownTimeout time.Duration
}

// GRPCConfig содержит параметры основного gRPC-сервера.
type GRPCConfig struct {
	Address         string
	Reflection      bool
	MaxReceiveBytes int
	MaxSendBytes    int
}

// HTTPConfig содержит параметры служебного HTTP-сервера.
type HTTPConfig struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

// LogConfig описывает минимальный уровень сообщений zap.
type LogConfig struct {
	Level string
}

// lookupEnv описывает функцию чтения переменной окружения.
// Отдельный тип позволяет тестировать конфигурацию без изменения окружения ОС.
type lookupEnv func(string) (string, bool)

// Load читает конфигурацию из переменных окружения и проверяет её целиком.
func Load() (Config, error) {
	return load(os.LookupEnv)
}

// load выполняет основную загрузку через переданную функцию чтения окружения.
func load(lookup lookupEnv) (Config, error) {
	// Сначала читаем окружение, потому что от него зависят безопасные defaults.
	environment := readString(lookup, "AUTH_ENV", defaultEnvironment)

	// Reflection и подробные debug-логи удобны локально, но выключены в production.
	isLocal := environment == "local" || environment == "development"
	defaultLogLevel := "info"
	if isLocal {
		defaultLogLevel = "debug"
	}

	reflection, err := readBool(lookup, "AUTH_GRPC_REFLECTION", isLocal)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := readDuration(lookup, "AUTH_SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	if err != nil {
		return Config{}, err
	}

	readHeaderTimeout, err := readDuration(lookup, "AUTH_HTTP_READ_HEADER_TIMEOUT", defaultReadHeaderTime)
	if err != nil {
		return Config{}, err
	}

	readTimeout, err := readDuration(lookup, "AUTH_HTTP_READ_TIMEOUT", defaultReadTime)
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := readDuration(lookup, "AUTH_HTTP_WRITE_TIMEOUT", defaultWriteTime)
	if err != nil {
		return Config{}, err
	}

	idleTimeout, err := readDuration(lookup, "AUTH_HTTP_IDLE_TIMEOUT", defaultIdleTime)
	if err != nil {
		return Config{}, err
	}

	maxReceiveBytes, err := readPositiveInt(lookup, "AUTH_GRPC_MAX_RECEIVE_BYTES", defaultMaxMessageBytes)
	if err != nil {
		return Config{}, err
	}

	maxSendBytes, err := readPositiveInt(lookup, "AUTH_GRPC_MAX_SEND_BYTES", defaultMaxMessageBytes)
	if err != nil {
		return Config{}, err
	}

	// Собираем единый объект, который дальше передаётся через явные зависимости.
	config := Config{
		Environment: environment,
		GRPC: GRPCConfig{
			Address:         readString(lookup, "AUTH_GRPC_ADDRESS", defaultGRPCAddress),
			Reflection:      reflection,
			MaxReceiveBytes: maxReceiveBytes,
			MaxSendBytes:    maxSendBytes,
		},
		HTTP: HTTPConfig{
			Address:           readString(lookup, "AUTH_HTTP_ADDRESS", defaultHTTPAddress),
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		Log: LogConfig{
			Level: readString(lookup, "AUTH_LOG_LEVEL", defaultLogLevel),
		},
		ShutdownTimeout: shutdownTimeout,
	}

	// Валидация выполняется один раз до запуска любых goroutine.
	if err := config.Validate(); err != nil {
		return Config{}, err
	}

	return config, nil
}

// Validate проверяет взаимные ограничения уже собранной конфигурации.
func (config Config) Validate() error {
	// Ограниченный список не позволяет случайной опечатке включить local-настройки.
	switch config.Environment {
	case "local", "development", "staging", "production":
	default:
		return fmt.Errorf("AUTH_ENV: неподдерживаемое окружение %q", config.Environment)
	}

	if err := validateAddress("AUTH_GRPC_ADDRESS", config.GRPC.Address); err != nil {
		return err
	}
	if err := validateAddress("AUTH_HTTP_ADDRESS", config.HTTP.Address); err != nil {
		return err
	}
	if config.GRPC.Address == config.HTTP.Address {
		return fmt.Errorf("gRPC и HTTP серверы не могут слушать один адрес %q", config.GRPC.Address)
	}

	// Поддерживаем только уровни, которые действительно нужны сервису.
	switch strings.ToLower(config.Log.Level) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("AUTH_LOG_LEVEL: неподдерживаемый уровень %q", config.Log.Level)
	}

	if config.GRPC.MaxReceiveBytes <= 0 || config.GRPC.MaxSendBytes <= 0 {
		return fmt.Errorf("ограничения размера gRPC-сообщений должны быть положительными")
	}
	if config.ShutdownTimeout <= 0 {
		return fmt.Errorf("AUTH_SHUTDOWN_TIMEOUT должен быть больше нуля")
	}
	if config.HTTP.ReadHeaderTimeout <= 0 || config.HTTP.ReadTimeout <= 0 ||
		config.HTTP.WriteTimeout <= 0 || config.HTTP.IdleTimeout <= 0 {
		return fmt.Errorf("таймауты HTTP-сервера должны быть больше нуля")
	}

	return nil
}

// readString возвращает явно заданное значение или безопасный default.
func readString(lookup lookupEnv, key, fallback string) string {
	value, exists := lookup(key)
	if !exists {
		return fallback
	}
	return strings.TrimSpace(value)
}

// readBool разбирает логическую переменную окружения.
func readBool(lookup lookupEnv, key string, fallback bool) (bool, error) {
	value, exists := lookup(key)
	if !exists {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, fmt.Errorf("%s: ожидается true или false: %w", key, err)
	}
	return parsed, nil
}

// readDuration разбирает значения наподобие 500ms, 10s или 2m.
func readDuration(lookup lookupEnv, key string, fallback time.Duration) (time.Duration, error) {
	value, exists := lookup(key)
	if !exists {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s: неверная длительность: %w", key, err)
	}
	return parsed, nil
}

// readPositiveInt разбирает положительное целое число.
func readPositiveInt(lookup lookupEnv, key string, fallback int) (int, error) {
	value, exists := lookup(key)
	if !exists {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s: ожидается положительное целое число", key)
	}
	return parsed, nil
}

// validateAddress проверяет TCP-адрес и допустимый диапазон порта.
func validateAddress(key, address string) error {
	_, portText, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%s: ожидается адрес вида host:port: %w", key, err)
	}

	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("%s: порт должен находиться в диапазоне 1..65535", key)
	}
	return nil
}

// Общая логика файла:
//  1. Load читает настройки и применяет только безопасные defaults.
//  2. Значения сложных типов преобразуются с понятной ошибкой запуска.
//  3. Validate проверяет отдельные поля и их взаимные ограничения.
//  4. Остальной код получает готовый Config и больше не читает os.Getenv.
