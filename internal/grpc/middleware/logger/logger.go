package logger

import (
	"context"
	"github.com/google/uuid"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

const RequestIDKey = "x-request-id"

func GetRequestID(ctx context.Context) string {
	reqID, ok := ctx.Value(RequestIDKey).(string)
	if !ok {
		reqID = "0"
	}

	return reqID
}

func NewUnary(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		reqID := uuid.New().String()
		ctx = context.WithValue(ctx, RequestIDKey, reqID)

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		statusCode := status.Code(err)

		log.Info("gRPC unary event ended",
			slog.String("requestID", reqID),
			slog.String("method", info.FullMethod),
			slog.String("code", statusCode.String()),
			slog.Duration("duration", duration),
		)

		return resp, err
	}
}

func NewStream(log *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		reqID := uuid.New().String()
		ctx := context.WithValue(ss.Context(), RequestIDKey, reqID)

		wrapStream := &wrapServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		log.Info("gRPC new conn",
			slog.String("requestID", reqID),
			slog.String("method", info.FullMethod),
		)

		err := handler(srv, wrapStream)

		duration := time.Since(start)
		statusCode := status.Code(err)

		log.Info("gRPC conn closed",
			slog.String("requestID", reqID),
			slog.String("method", info.FullMethod),
			slog.String("code", statusCode.String()),
			slog.String("session time", duration.String()),
		)

		return err
	}
}

type wrapServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrapServerStream) Context() context.Context {
	return w.ctx
}
