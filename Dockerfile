# Build stage
FROM m.daocloud.io/docker.io/golang:1.24-alpine AS builder

WORKDIR /app

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 go build -o mcp-milvus ./cmd/mcp-milvus

# Runtime stage
FROM m.daocloud.io/docker.io/alpine:latest

# Install ca-certificates for HTTPS requests
RUN sed -i 's|dl-cdn.alpinelinux.org|mirrors.aliyun.com|g' /etc/apk/repositories && \
    apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/mcp-milvus .

# Expose the port the app runs on
EXPOSE 8080

# Command to run the executable
ENTRYPOINT ["./mcp-milvus"]