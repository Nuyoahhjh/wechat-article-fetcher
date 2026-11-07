FROM golang:1.23-alpine AS builder
WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:3.20
RUN apk --no-cache add ca-certificates wget
WORKDIR /app
COPY --from=builder /server /app/server
RUN mkdir -p /data
EXPOSE 8080
CMD ["/app/server"]
