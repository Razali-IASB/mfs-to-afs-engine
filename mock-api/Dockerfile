FROM golang:1.21-alpine AS builder

WORKDIR /build
COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-receiver main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /build/api-receiver .

EXPOSE 3001

CMD ["./api-receiver"]
