FROM golang:1.25-alpine AS builder

RUN apk add make

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN make build

FROM alpine:3.18

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY .env.local .
COPY ./docs/ ./docs/

COPY --from=builder /app/gopher .

EXPOSE 8080
EXPOSE 6060

ENV CONFIG_PATH /app/.env.local

CMD ["./gopher"]