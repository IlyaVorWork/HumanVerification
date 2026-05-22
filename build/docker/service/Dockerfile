FROM golang:1.25.5-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/service ./cmd/app

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /bin/service /app/service
COPY --from=builder /src/boot.yaml /app/boot.yaml
COPY --from=builder /src/docs /app/docs
ENTRYPOINT ["/app/service"]
