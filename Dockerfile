FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 构建
RUN CGO_ENABLED=0 go build -o server .

FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/server .

COPY --from=builder /app/templates ./templates

EXPOSE 8080

CMD ["./server"]