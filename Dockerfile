FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /klyntar-server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /klyntar-server /klyntar-server
COPY --from=builder /app/migrations /migrations
EXPOSE 50051
ENTRYPOINT ["/klyntar-server"]
