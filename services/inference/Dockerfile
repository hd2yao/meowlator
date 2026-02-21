FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /out/inference ./cmd/inference

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/inference /app/inference
EXPOSE 8081
CMD ["/app/inference"]
