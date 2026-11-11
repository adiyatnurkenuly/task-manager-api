
FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /server ./cmd/api

FROM alpine:3.24

RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /server /server

USER app

EXPOSE 8080

CMD ["/server"]
