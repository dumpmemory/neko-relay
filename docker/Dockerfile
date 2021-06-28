FROM golang:1.16 as builder

# Set Environment Variables
ENV HOME /app
ENV CGO_ENABLED 0
ENV GOOS linux

WORKDIR /app
COPY go.mod go.sum ./
RUN go get -u
COPY . .

# Build the Go app
RUN go build -ldflags='-s -w' -v -a -installsuffix cgo -o neko-relay

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /bin/

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/neko-relay .

ENTRYPOINT ["/bin/neko-relay"]