FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY ./backend-api/go.mod ./
COPY ./backend-api/main.go ./

RUN go mod download && \
  go build -o backend-api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/backend-api .
EXPOSE 8080
CMD ["./backend-api"]
