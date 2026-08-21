package interceptor

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"   // зачем эта библиотека? что такое peer?
	"google.golang.org/grpc/status" // зачем эта библиотека?
)

// Функцию, которую gRPC будет вызывать для каждого unary RPC.
// UnaryLogging записывает один структурированный log event после unary RPC.
func UnaryLogging(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo, // Содержит информацию о streaming RPC
		handler grpc.UnaryHandler, // настоящий обработчик RPC (привычный handler)
	) (response any, err error) {
		startedAt := time.Now()
		response, err = handler(ctx, request)
		logCompletedCall(logger, ctx, info.FullMethod, startedAt, err)
		return response, err
	}
}

// StreamLogging записывает результат streaming RPC после закрытия потока.
func StreamLogging(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(
		server any, // Это объект, реализующий gRPC-сервис.
		stream grpc.ServerStream, // Это объект серверного потока.
		info *grpc.StreamServerInfo, // Содержит информацию о streaming RPC
		handler grpc.StreamHandler,
	) error {
		startedAt := time.Now()
		err := handler(server, stream)
		logCompletedCall(logger, stream.Context(), info.FullMethod, startedAt, err)
		return err
	}
}

// Выбирает уровень лога по итоговому gRPC status code и 
// отправляет сам лог с полной информацией о запросе
func logCompletedCall(
	logger *zap.Logger,
	ctx context.Context,
	method string,
	startedAt time.Time,
	err error,
) {
	code := status.Code(err)
	fields := []zap.Field{
		zap.String("request_id", RequestIDFromContext(ctx)),
		zap.String("grpc_method", method),
		zap.String("grpc_status_code", code.String()),
		zap.Duration("duration", time.Since(startedAt)),
		zap.String("peer_address", peerAddress(ctx)),
	}

	switch code {
	case codes.Internal, codes.Unknown, codes.DataLoss:
		logger.Error("gRPC запрос завершен", fields...)
	case codes.Unavailable, codes.ResourceExhausted, codes.DeadlineExceeded:
		logger.Warn("gRPC запрос завершен", fields...)
	default:
		logger.Info("gRPC запрос завершен", fields...)
	}
}

// Безопасно извлекает сетевой адрес gRPC-клиента.
func peerAddress(ctx context.Context) string {
	remotePeer, ok := peer.FromContext(ctx)
	if !ok || remotePeer.Addr == nil {
		return "unknown"
	}

	return remotePeer.Addr.String()
}

// Общая логика файла:
//  1. До handler фиксируется время начала вызова.
//  2. После handler вычисляются длительность и итоговый gRPC code.
//  3. Технические ошибки получают warn/error, а ожидаемые клиентские — info.
//  4. Пароли и protobuf payload никогда не записываются в лог.

/*
Это объект серверного потока.
Через него можно:
- получать сообщения от клиента;
- отправлять сообщения клиенту;
- получить контекст RPC;
- работать с headers и trailers.

======================================

Основные методы интерфейса выглядят примерно так:

type ServerStream interface {
	Context() context.Context
	SendMsg(message any) error
	RecvMsg(message any) error
	SetHeader(metadata.MD) error
	SendHeader(metadata.MD) error
	SetTrailer(metadata.MD)

=====================================

Клиент открыл stream
    ↓
Обмен сообщениями 10 минут
    ↓
Клиент закрыл stream
    ↓
handler вернул управление

}
*/