FROM golang:1.23.3-alpine AS builder
WORKDIR /app
COPY . .
RUN GOOS=linux GOARCH=$TARGETARCH go build -ldflags="-w -s" -o build/proxy-pool main.go

FROM alpine:latest
RUN apk add --no-cache tzdata
WORKDIR /app
COPY --from=builder /app/build/proxy-pool .
RUN mkdir -p /etc/proxy-pool
VOLUME /etc/proxy-pool
ENV PROXY_SERVER_DEBUG=false
ENV TZ=Asia/Shanghai
EXPOSE 8001
CMD chmod +x ./proxy-pool && ./proxy-pool -config /etc/proxy-pool/config.json -port 8001 -host 0.0.0.0