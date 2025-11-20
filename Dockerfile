FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o pqgb .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/pqgb /
COPY static /static
EXPOSE 8080
CMD ["/pqgb"]