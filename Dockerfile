FROM golang:1.22.5 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server


FROM alpine:3.20

RUN apk add --no-cache ca-certificates \
	&& adduser -D -H -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/server ./server

USER appuser

EXPOSE 8080

CMD ["./server"]
