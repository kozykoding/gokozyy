FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY backend ./backend
WORKDIR /app/backend
RUN go build -o server .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/backend/server /app/server
CMD ["./server"]
