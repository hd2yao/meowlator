FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /out/api ./cmd/api

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/api /app/api
EXPOSE 8080
CMD ["/app/api"]
