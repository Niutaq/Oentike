FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o control-plane .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/control-plane .

EXPOSE 50051
CMD ["./control-plane"]
