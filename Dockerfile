FROM golang:1.15-alpine AS builder

# Set necessary environmet variables needed for our image
ENV CGO_ENABLED=1 \
    GOOS=linux

# add certificates
RUN apk add --no-cache ca-certificates 2>&1

# Move to working directory /build
WORKDIR /build

# Copy the code into the container
COPY . .

# Additional packages required
RUN apk -U add musl-dev gcc

# Build the application
RUN go build -v -a -tags netgo -ldflags '-w -extldflags "-static"' .

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
RUN cp /build/github2telegram .

# Build a small image
FROM scratch

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.pem
COPY --from=builder /dist/github2telegram /app

# Command to run
ENTRYPOINT ["/app/github2telegram"]
