# ---------- stage 1 : build go ----------
FROM golang:1.25.5-alpine
WORKDIR /humanVerification
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .

CMD ["go", "run", "./cmd/app/main.go" ]