FROM golang:1.22-alpine as builder

# Set environment variables to ensure cross-compilation
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /src
COPY go.sum go.sum
COPY go.mod go.mod
COPY main.go main.go

RUN go mod download
RUN go build -o restart-scheduler .

FROM alpine:3
WORKDIR /app
COPY --from=builder /src/restart-scheduler /app/restart-scheduler
CMD ["/app/restart-scheduler", "--env=inCluster"]
