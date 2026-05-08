FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /worker-task ./main.go

FROM gcr.io/distroless/static-debian12
COPY --from=builder /worker-task /worker-task
ENTRYPOINT ["/worker-task"]
