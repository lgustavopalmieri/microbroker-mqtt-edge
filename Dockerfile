FROM golang:1.25 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o microbroker ./cmd/main.go

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
RUN mkdir -p /data
COPY --from=builder /app/microbroker .
EXPOSE 1883
ENTRYPOINT ["/app/microbroker"]
