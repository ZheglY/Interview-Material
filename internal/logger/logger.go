package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const serviceName = "auth-service"

// Создает логер с console форматом локально и JSON в production
func New(environment, levelText string) (*zap.Logger, error) {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(strings.ToLower(levelText))); err != nil { // "info"  → zapcore.InfoLevel
		return nil, fmt.Errorf("не удалось разобраться уровень логирования %q: %w", levelText, err)
	}

	var config zap.Config
	if environment == "local" || environment == "development" {
		// Development-конфигурация в консоли для удобства разработки
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		// Production-конфигурация пишет JSON, пригодный для Loki или Elasticsearch.
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	}

	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.Level = zap.NewAtomicLevelAt(level)

	// После Build() получается рабочий объект - *zap.Logger
	// Stack trace показывает цепочку вызовов, которая привела к месту записи ошибки.Только для уровня error
	newLogger, err := config.Build(zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		return nil, fmt.Errorf("создать zap логер: %w", err)
	}

	return newLogger.With(zap.String("service", serviceName)), nil // Возвращается готовый логер c названием сервиса "service": "auth-service"
}


// Общая логика файла:
//  1. Уровень логирования проверяется до запуска приложения.
//  2. Локально сообщения удобны для чтения, а в production имеют JSON-формат.
//  3. Каждый log event автоматически получает имя сервиса.
//  4. Пароли, токены и тела запросов в базовые поля логера не добавляются.

/*
zap.Config в себе содержит конфигурацию логера:
формат вывода;
уровень логирования;
названия JSON-полей;
формат времени;
место вывода;
правила вывода stack trace.

debug < info < warn < error < dpanic < panic < fatal
*/