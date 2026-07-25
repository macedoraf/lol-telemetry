# Dockerfile for the mock Live Client Data API used by the Docker Compose test stack.
FROM golang:1.22-alpine

WORKDIR /app

COPY test/mocksrv/go.mod .
COPY test/mocksrv/main.go .
RUN go mod tidy && go build -o mocksrv .

ENV MOCK_PAYLOAD=/app/mocks/allgamedata.json

EXPOSE 2999 8080

CMD ["./mocksrv"]
