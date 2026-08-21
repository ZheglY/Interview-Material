package interceptor

import (
	"context"
	"fmt"
	"runtime/debug"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryRecovery не позволяет panic внутри unary handler завершить весь процесс.
func UnaryRecovery(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (response any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logPanic(logger, ctx, info.FullMethod, recovered)
				err = status.Error(codes.Internal, "внутренняя ошибка сервера")
			}
		}()

		return handler(ctx, request)
	}
}

// StreamRecovery не позволяет panic внутри streaming handler завершить процесс.
func StreamRecovery(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logPanic(logger, stream.Context(), info.FullMethod, recovered)
				err = status.Error(codes.Internal, "внутренняя ошибка сервера")
			}
		}()

		return handler(server, stream)
	}
}

// logPanic сохраняет технические сведения для расследования, но не отдаёт их клиенту.
func logPanic(logger *zap.Logger, ctx context.Context, method string, recovered any) {
	logger.Error(
		"перехвачен panic в gRPC handler",
		zap.String("request_id", RequestIDFromContext(ctx)),
		zap.String("grpc_method", method),
		// Значение panic не пишется: оно может случайно содержать пароль или токен.
		zap.String("panic_type", fmt.Sprintf("%T", recovered)),
		zap.ByteString("stack_trace", debug.Stack()), // текстовый stack trace текущей goroutine
	)
}



// Общая логика файла:
//  1. defer/recover перехватывает panic на границе transport-слоя.
//  2. Полный stack trace остаётся только во внутренних логах.
//  3. Клиент получает нейтральный codes.Internal без деталей реализации.