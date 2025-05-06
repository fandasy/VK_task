FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o pubsubapp ./cmd/pubsub-server/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/pubsubapp .

COPY --from=builder /app/config/prod.yaml config/default.yaml

ENV CONFIG_PATH=config/default.yaml

EXPOSE 8082

RUN chmod +x ./pubsubapp

CMD ["./pubsubapp"]
