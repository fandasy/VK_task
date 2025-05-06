package pubsub

import (
	"context"
	"errors"
	"log/slog"

	"VK_task/internal/grpc/middleware/logger"
	"VK_task/internal/pkg/logger/sl"
	pb "VK_task/pkg/api/pubsub"
	"VK_task/pkg/e"
	sp "VK_task/pkg/subpub"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service struct {
	pb.UnimplementedPubSubServer
	ps  sp.SubPub
	log *slog.Logger

	srvStop <-chan struct{}
}

func New(ps sp.SubPub, log *slog.Logger, stop <-chan struct{}) *Service {
	return &Service{
		ps:      ps,
		log:     log,
		srvStop: stop,
	}
}

func (s *Service) Subscribe(req *pb.SubscribeRequest, stream pb.PubSub_SubscribeServer) error {
	log := s.log.With(
		slog.String("requestID", logger.GetRequestID(stream.Context())),
	)

	log.Debug("Conn data", slog.String("key", req.Key))

	if req.Key == "" {
		log.Warn("Req.Key is empty")

		return status.Error(codes.InvalidArgument, "key required")
	}

	errCh := make(chan error)
	defer close(errCh)

	handler := func(msg interface{}) {
		data, ok := msg.(string)
		if !ok {
			return
		}

		event := &pb.Event{
			Data: data,
		}

		if err := stream.Send(event); err != nil {
			errCh <- err
		}
	}

	sub, err := s.ps.Subscribe(req.Key, handler)
	if err != nil {
		log.Error("SubPub Subscribe operation failed", sl.Err(err))

		return status.Error(codes.Internal, e.String("failed to subscribe", err))
	}
	defer sub.Unsubscribe()

	select {
	case err := <-errCh:
		log.Error("Send event to stream failed", sl.Err(err))

		return status.Error(codes.Unavailable, e.String("failed to send event", err))

	case <-stream.Context().Done():
		// log.Info("Stream context is done")

		return status.FromContextError(stream.Context().Err()).Err()

	case <-s.srvStop:
		return status.Error(codes.Canceled, "Server stopping")
	}
}

func (s *Service) Publish(ctx context.Context, req *pb.PublishRequest) (*emptypb.Empty, error) {
	log := s.log.With(
		slog.String("requestID", logger.GetRequestID(ctx)),
	)

	log.Debug("Request data",
		slog.String("key", req.Key),
		slog.String("data", req.Data),
	)

	if req.Key == "" {
		log.Warn("Req.Key is empty")

		return nil, status.Error(codes.InvalidArgument, "key required")
	}
	if req.Data == "" {
		log.Warn("Req.Data is empty")

		return nil, status.Error(codes.InvalidArgument, "data required")
	}

	if err := ctx.Err(); err != nil {
		log.Error("Request context is done")

		return nil, status.FromContextError(err).Err()
	}

	if err := s.ps.Publish(req.Key, req.Data); err != nil {
		if errors.Is(err, sp.ErrNoSuchSubject) {
			log.Warn("SubPub no such subject", slog.String("subject", req.Key))

			return nil, status.Error(codes.InvalidArgument, "no such subject")
		}

		log.Error("SubPub Publish operation failed", sl.Err(err))

		return nil, status.Error(codes.Internal, e.String("failed to publish", err))
	}
	return &emptypb.Empty{}, nil
}
