# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install protoc and go plugins if generation was needed, 
# but we assume the proto code is already generated in the source tree 
# or we generate it here if needed.
# Since we can't easily run protoc in alpine without apk, let's assume pre-generated.

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o kv-server ./cmd/server/main.go

# Run stage
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/kv-server .

EXPOSE 50051

ENTRYPOINT ["./kv-server"]
