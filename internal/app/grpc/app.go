package grpcapp

import (
	"VK_task/internal/grpc/middleware/logger"
	pb "VK_task/pkg/api/pubsub"
	"VK_task/pkg/e"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
)

type App struct {
	gRPCServer *grpc.Server
	addr       string
	log        *slog.Logger

	stop chan struct{}
}

func New(ip string, port int, log *slog.Logger, service pb.PubSubServer, stop chan struct{}) *App {
	gRPCServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(logger.NewUnary(log)),
		grpc.ChainStreamInterceptor(logger.NewStream(log)),
	)
	pb.RegisterPubSubServer(gRPCServer, service)

	return &App{
		gRPCServer: gRPCServer,
		addr:       fmt.Sprintf("%s:%d", ip, port),
		log:        log,
		stop:       stop,
	}
}

func (app *App) Start() error {
	l, err := net.Listen("tcp", app.addr)
	if err != nil {
		return e.Wrap("net listen failed", err)
	}

	app.log.Info("Net listen tcp", slog.String("addr", l.Addr().String()))

	if err := app.gRPCServer.Serve(l); err != nil {
		return e.Wrap("gRPC server serve failed", err)
	}

	return nil
}

func (app *App) Stop() {
	select {
	case _, ok := <-app.stop:
		if !ok {
			return
		}
	default:

		close(app.stop)

		app.gRPCServer.GracefulStop()
	}
}
