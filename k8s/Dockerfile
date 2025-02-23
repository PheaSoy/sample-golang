# Use the official Go image
FROM golang:1.20 as builder

WORKDIR /app

# Copy the Go source code
COPY . .

# Build the Go app
RUN go mod tidy
RUN go build -o k8s-watch-event .

# Start a new image and copy the built app from the builder image
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/k8s-watch-event .

# Install any required packages (e.g., bash, curl)
RUN apk --no-cache add bash

CMD ["./k8s-watch-event"]
