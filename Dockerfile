FROM golang:1.25.5-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /out/api ./api

EXPOSE 8080
CMD ["./api"]
