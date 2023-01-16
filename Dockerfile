# Build stage
FROM golang:1.19.5-alpine3.17 AS builder
WORKDIR /app
COPY . .
RUN go build -o main cmd/server/main.go

# Run stage
FROM alpine:3.17
WORKDIR /app
COPY --from=builder /app/main .
COPY ./cert ./cert

VOLUME [ "/app/img" ]
EXPOSE 8080
CMD ["/app/main"]