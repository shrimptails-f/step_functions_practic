FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN go mod tidy
RUN go build -o /out/parent-task ./main.go

FROM alpine:3.20
RUN adduser -D -u 10001 appuser
USER appuser
WORKDIR /app
COPY --from=build /out/parent-task /app/parent-task
ENTRYPOINT ["/app/parent-task"]
