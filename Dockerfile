FROM golang:1.20-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod tidy

COPY . .

RUN go build -o /build/main ./cmd

FROM alpine:latest

WORKDIR /app

COPY --from=builder /build/main .

EXPOSE 3040

CMD ["./main"]
