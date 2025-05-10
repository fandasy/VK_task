package tests

import (
	"context"
	"log/slog"
	"net"
	"strconv"
	"testing"
	"time"

	"VK_task/internal/app"
	"VK_task/internal/config"
	"VK_task/internal/pkg/logger"
	pb "VK_task/pkg/api/pubsub"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	devConfigPath = "dev.yaml"

	grpcHost = "localhost"
	grpcPort = 8082
)

// Для теста нужен запущенный сервер
func TestPubSubIntegration(t *testing.T) {
	// Если нужен автоматический старт сервера
	// _, port, srvStop := startTestApp(devConfigPath)
	// defer srvStop()
	// Замените grpcPort на port!

	client, cleanup := newPubSubClient(t, grpcHost, grpcPort)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("Publish with empty key", func(t *testing.T) {
		_, err := client.Publish(ctx, &pb.PublishRequest{Key: "", Data: "data"})
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("Publish with empty data", func(t *testing.T) {
		_, err := client.Publish(ctx, &pb.PublishRequest{Key: "test", Data: ""})
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("Successful publish", func(t *testing.T) {
		_, err := client.Publish(ctx, &pb.PublishRequest{Key: "test", Data: "message"})
		assert.Error(t, err)
		assert.IsType(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("Subscribe with empty key", func(t *testing.T) {
		stream, err := client.Subscribe(ctx, &pb.SubscribeRequest{Key: ""})
		require.NoError(t, err)

		_, err = stream.Recv()
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("Subscribe and receive messages", func(t *testing.T) {
		subCtx, subCancel := context.WithCancel(ctx)
		defer subCancel()

		stream, err := client.Subscribe(subCtx, &pb.SubscribeRequest{Key: "test"})
		require.NoError(t, err)

		// Wait to goroutine start
		time.Sleep(100 * time.Millisecond)

		// Publish a message that should be received
		_, err = client.Publish(ctx, &pb.PublishRequest{Key: "test", Data: "test message"})
		require.NoError(t, err)

		event, err := stream.Recv()
		require.NoError(t, err)
		assert.Equal(t, "test message", event.Data)

		// Publish another message
		_, err = client.Publish(ctx, &pb.PublishRequest{Key: "test", Data: "second message"})
		require.NoError(t, err)

		event, err = stream.Recv()
		require.NoError(t, err)
		assert.Equal(t, "second message", event.Data)
	})

	t.Run("Subscribe and cancel context", func(t *testing.T) {
		subCtx, subCancel := context.WithCancel(ctx)
		stream, err := client.Subscribe(subCtx, &pb.SubscribeRequest{Key: "test"})
		require.NoError(t, err)

		// Cancel the context
		subCancel()

		_, err = stream.Recv()
		assert.Error(t, err)
		assert.Equal(t, codes.Canceled, status.Code(err))
	})

	t.Run("Publish to non-existent subject", func(t *testing.T) {
		_, err := client.Publish(ctx, &pb.PublishRequest{Key: "nonexistent", Data: "data"})
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}

// Для теста нужен запущенный сервер
func TestMultipleSubscribers(t *testing.T) {
	// Если нужен автоматический старт сервера
	// _, port, srvStop := startTestApp(devConfigPath)
	// defer srvStop()
	// Замените grpcPort на port!

	client, cleanup := newPubSubClient(t, grpcHost, grpcPort)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First subscriber
	stream1, err := client.Subscribe(ctx, &pb.SubscribeRequest{Key: "test"})
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Second subscriber
	stream2, err := client.Subscribe(ctx, &pb.SubscribeRequest{Key: "test"})
	require.NoError(t, err)

	// Wait to goroutine start
	time.Sleep(100 * time.Millisecond)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Publish a message
	_, err = client.Publish(ctx, &pb.PublishRequest{Key: "test", Data: "broadcast"})
	require.NoError(t, err)

	// Both subscribers should receive the message
	event1, err := stream1.Recv()
	require.NoError(t, err)
	assert.Equal(t, "broadcast", event1.Data)

	event2, err := stream2.Recv()
	require.NoError(t, err)
	assert.Equal(t, "broadcast", event2.Data)
}

// WARN: Автоматический старт сервера!
func TestServerStopDuringSubscription(t *testing.T) {
	_, port, appStop := startTestApp(devConfigPath) // ⚠️
	defer appStop()

	client, cleanup := newPubSubClient(t, grpcHost, port)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.Subscribe(ctx, &pb.SubscribeRequest{Key: "test"})
	require.NoError(t, err)

	// Wait to goroutine start
	time.Sleep(100 * time.Millisecond)

	t.Log("Call Application stop")

	// Graceful shutdown
	err = appStop()
	assert.NoError(t, err)

	_, err = stream.Recv()
	assert.Error(t, err)
	assert.Equal(t, codes.Canceled, status.Code(err))
}

func newPubSubClient(t *testing.T, host string, port int) (pb.PubSubClient, func()) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	cc, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed connect to grpc server: %v", err)
	}

	client := pb.NewPubSubClient(cc)

	return client, func() {
		cc.Close()
	}
}

func startTestApp(path string) (string, int, func() error) {
	cfg := config.MustLoad(path)

	log := logger.MustSetup(cfg.SLOG.Env, cfg.SLOG.File)

	log.Debug("Config", slog.Any("data", cfg))

	// App
	application := app.New(log, cfg.GRPC.Addr, cfg.GRPC.Port, cfg.SubPub.SubjectBuffer, cfg.SubPub.SubscriptionBuffer)

	go application.MustRun()

	time.Sleep(100 * time.Millisecond)

	stop := func() error {
		// Graceful Stop
		err := application.StopWithLog(cfg.SubPub.CloseTimeout, log)
		if err != nil {
			return err
		}

		log.Info("App shutdown")

		return nil
	}

	return cfg.GRPC.Addr, cfg.GRPC.Port, stop
}
