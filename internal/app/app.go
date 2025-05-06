package app

import (
	"context"
	"log/slog"
	"time"

	grpcapp "VK_task/internal/app/grpc"
	"VK_task/internal/grpc/handler/pubsub"
	"VK_task/pkg/e"
	"VK_task/pkg/subpub"
)

type App struct {
	GRPCApp *grpcapp.App
	SubPub  subpub.SubPub
}

func New(log *slog.Logger,
	grpcSrvIP string,
	grpcSrvPort int,
	subjectBuffer int,
) *App {
	subPub := subpub.NewSubPub(subpub.Config{
		SubjectPuffer: subjectBuffer,
	})

	// Для GracefulStop
	grpcStopCh := make(chan struct{})

	PubSubService := pubsub.New(subPub, log, grpcStopCh)

	grpcApp := grpcapp.New(grpcSrvIP, grpcSrvPort, log, PubSubService, grpcStopCh)

	return &App{
		GRPCApp: grpcApp,
		SubPub:  subPub,
	}
}

func (app *App) MustRun() {
	if err := app.Run(); err != nil {
		panic(e.Wrap("App starting failed", err))
	}
}

func (app *App) Run() error {
	if err := app.GRPCApp.Start(); err != nil {
		return e.Wrap("grpc application startup failed", err)
	}

	return nil
}

func (app *App) Stop(spCloseTimeout time.Duration) error {
	// С начало жду завершение handler которые могут использовать subPub
	app.GRPCApp.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), spCloseTimeout)
	defer cancel()

	if err := app.SubPub.Close(ctx); err != nil {
		return e.Wrap("Sub/Pub close failed", err)
	}

	return nil
}

func (app *App) StopWithLog(spCloseTimeout time.Duration, log *slog.Logger) error {
	log.Info("Stopping gRPC server")

	// С начало жду завершение handler которые могут использовать subPub
	app.GRPCApp.Stop()

	log.Info("gRPC server stopped")

	ctx, cancel := context.WithTimeout(context.Background(), spCloseTimeout)
	defer cancel()

	log.Info("Closing the subPub event bus")

	if err := app.SubPub.Close(ctx); err != nil {
		return e.Wrap("Sub/Pub close failed", err)
	}

	log.Info("SubPub event bus closed")

	return nil
}
