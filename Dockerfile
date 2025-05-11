FROM golang:1.22.3 AS builder
WORKDIR /app
COPY go.mod ./
# COPY go.sum ./ não utilizamos pois usamos apenas libs nativas
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mqtt-client .

FROM scratch
WORKDIR /app
COPY --from=builder /app/mqtt-client .
EXPOSE 6081
ENTRYPOINT ["/app/mqtt-client"]
