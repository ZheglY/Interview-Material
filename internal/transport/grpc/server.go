// Package grpcserver создаёт и управляет основным gRPC-сервером приложения.
package grpcserver

import (
	"context"
	"net"
	"time"

	"github.com/ZheglY/Interview-Material/internal/config"
	"github.com/ZheglY/Interview-Material/internal/transport/grpc/interceptor"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// Server объединяет gRPC runtime и стандартный health service.
type Server struct {
	grpc *grpc.Server
	health *health.Server // почему казатели?
}

// New создаёт gRPC-сервер, но ещё не открывает сетевой listener.
func New(cfg config.GRPCConfig, logger *zap.Logger) *Server {

	// // Interceptor-ы объявлены в порядке внешней обёртки к handler-у.
	options := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			interceptor.UnaryRequestID(),
			interceptor.UnaryLogging(logger),
			interceptor.UnaryRecovery(logger),
		),
		grpc.ChainStreamInterceptor(
			interceptor.StreamRequestID(),
			interceptor.StreamLogging(logger),
			interceptor.StreamRecovery(logger),
		),

		grpc.MaxRecvMsgSize(cfg.MaxReceiveBytes), // Устанавливает максимальный размер одного сообщения, которое сервер разрешает получить от клиента.
		grpc.MaxSendMsgSize(cfg.MaxSendBytes), // Устанавливает максимальный размер одного сообщения, которое сервер разрешает отправить клиенту.
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute, // Если у соединения нет активных RPC в течение пяти минут, сервер начинает его аккуратно закрывать
			Time:              2 * time.Hour, // Если по соединению достаточно долго нет сетевой активности, сервер отправляет HTTP/2 PING, чтобы проверить, жива ли другая сторон
			Timeout:           20 * time.Second, // После отправки PING сервер ждёт признаков активности от клиента не более 20 секунд.		
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime: 10 * time.Second, // Клиент не должен отправлять keepalive PING чаще одного раза в десять секунд.
			PermitWithoutStream: false, // Сервер не разрешает клиенту отправлять keepalive PING, если на соединении нет активного RPC/stream.
		}),
	}

	// Создаётся объект gRPC-сервера со всеми собранными настройками
	grpcServer := grpc.NewServer(options...)
	// служебный gRPC-сервис с методами для проверки работоспособности
	healthServer := health.NewServer()

	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING) // Устанавливается состояние: NOT_SERVING

	// Подключает реализацию health-check к основному gRPC-серверу.
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	if cfg.Reflection {
		reflection.Register(grpcServer)
	}

	return &Server{
		grpc: grpcServer,
		health: healthServer,
	}
}

// Serve начинает принимать gRPC-запросы
func (server *Server) Serve(listener net.Listener) error {
	return server.grpc.Serve(listener)
}

// MarkServing переводит стандартный gRPC health service в состояние SERVING.
func (server *Server) MarkServing() {
	server.health.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
}
// GracefulStop прекращает приём новых RPC и даёт активным запросам завершиться.
func (server *Server) GracefulStop(ctx context.Context) {
	// Health меняется раньше остановки listener, чтобы балансировщик убрал экземпляр.
	server.health.Shutdown()

	finished := make(chan struct{})
	go func() {
		server.grpc.GracefulStop()
		close(finished)
	}()

	select {
	case <-finished:
		// Все активные RPC завершились в пределах shutdown timeout.
	case <-ctx.Done():
		// Жёсткая остановка не позволяет зависшему RPC удерживать deployment бесконечно.
		server.grpc.Stop()
		<-finished
	}
}

/*
Reflection — это механизм самоописания gRPC-сервера.

Без reflection клиенту обычно нужен .proto-файл или заранее сгенерированный клиентский код, чтобы знать:
- какие сервисы доступны;
- какие методы существуют;
- какие типы запросов принимаются;
- какие типы ответов возвращаются;
- как устроены protobuf-сообщения.
*/

// Общая логика файла:
//  1. New собирает transport options, interceptor-ы и health service.
//  2. Reflection включается только конфигурацией, а не безусловно.
//  3. MarkServing вызывается после запуска всех listeners.
//  4. GracefulStop сначала снимает readiness, затем ждёт активные RPC.