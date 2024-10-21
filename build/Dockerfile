FROM golang:1.22.1
WORKDIR /workspace
COPY . .

EXPOSE 9100

RUN CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -o /go/hercules-linux-arm64 ./cmd/hercules
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /go/hercules-linux-amd64 ./cmd/hercules

ENTRYPOINT []