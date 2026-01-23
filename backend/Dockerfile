FROM golang:1.24.1-alpine as builder

RUN apk add --no-cache gcc musl-dev sqlite

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o main .

FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /app

COPY --from=builder /app/main .
RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./main"]