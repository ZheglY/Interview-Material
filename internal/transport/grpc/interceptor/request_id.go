package interceptor

import (
	"context"
	"crypto/rand" // криптографически стойкий генератор случайных данных.
	"encoding/hex"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata" // что делает эта библиотека?
	"google.golang.org/grpc/status"
)

const (
	RequestIDMetadataKey = "x-request-id"

	// Ограничение не позволяет клиенту раздувать логи произвольной metadata.
	maxRequestIDLength = 128
)

// requestIDContextKey — приватный тип ключа, исключающий коллизии context values.
type requestIDContextKey struct{}

// UnaryRequestID добавляет request ID к каждому unary RPC.
func UnaryRequestID() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Валидный ID клиента сохраняет сквозную трассировку между сервисами.
		requestID, err := resolveRequestID(ctx)
		if err != nil {
			return nil, status.Error(codes.Internal, "не удалось создать request ID")
		}

		// Значение помещается в новый context и не изменяет исходный объект.
		ctx = context.WithValue(ctx, requestIDContextKey{}, requestID)

		// Клиент получает тот же ID в response headers для диагностики запроса.
		if err := grpc.SetHeader(ctx, metadata.Pairs(RequestIDMetadataKey, requestID)); err != nil {
			return nil, status.Error(codes.Internal, "не удалось установить request ID")
		}

		return handler(ctx, request)
	}
}

// StreamRequestID добавляет request ID к server/client streaming RPC.
func StreamRequestID() grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		requestID, err := resolveRequestID(stream.Context())
		if err != nil {
			return status.Error(codes.Internal, "не удалось создать request ID")
		}

		// Header устанавливается до вызова handler, пока ответ ещё не отправлен.
		if err := stream.SetHeader(metadata.Pairs(RequestIDMetadataKey, requestID)); err != nil {
			return status.Error(codes.Internal, "не удалось установить request ID")
		}

		// Обёртка меняет только Context, сохраняя остальные методы ServerStream.
		wrapped := &serverStreamWithContext{
			ServerStream: stream,
			ctx:          context.WithValue(stream.Context(), requestIDContextKey{}, requestID),
		}
		return handler(server, wrapped)
	}
}

// serverStreamWithContext позволяет передать обогащённый context streaming handler-у.
type serverStreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

// Context возвращает context с request ID вместо исходного context потока.
func (stream *serverStreamWithContext) Context() context.Context {
	return stream.ctx
}

func RequestIDFromContext(ctx context.Context) string {
	request_id, _ := ctx.Value(requestIDContextKey{}).(string)
	return request_id
}

/* 
функция пытается получить request ID, который клиент 
передал в gRPC metadata. Если корректного ID нет, 
сервер генерирует новый.

gRPC metadata — это служебные данные, передаваемые вместе с RPC-вызовом. 
Они похожи на HTTP-заголовки.
*/ 
func resolveRequestID(ctx context.Context) (string, error) {
	if incoming, ok := metadata.FromIncomingContext(ctx); ok {
		values := incoming.Get(RequestIDMetadataKey)
		if len(values) > 0 && isValidRequestID(values[0]) {
			return values[0], nil
		}
	}

	return generateRequestID() // возвращаем новое случайное id клиента 
}

// generateRequestID создаёт 128-битный случайный идентификатор в hex-формате.
func generateRequestID() (string, error) {
	randomBytes := make([]byte, 16) // слайс из 16 нулей

	// rand.Read(randomBytes) записывает случайные данные в уже выделенную память.
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("прочитать криптографическую случайность: %w", err)
	}
	return hex.EncodeToString(randomBytes), nil
}

// isValidRequestID разрешает только безопасные ASCII-символы для логов и metadata.
func isValidRequestID(value string) bool {
	if value == "" || len(value) > maxRequestIDLength {
		return false
	}

	for index := 0; index < len(value); index++ {
		character := value[index]
		isLetter := character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z'
		isDigit := character >= '0' && character <= '9'
		isSeparator := character == '-' || character == '_' || character == '.'
		if !isLetter && !isDigit && !isSeparator {
			return false
		}
	}
	return true
}