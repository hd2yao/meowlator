FROM golang:1.22-bookworm AS builder
WORKDIR /src
COPY go.mod ./
COPY go.sum ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /out/inference ./cmd/inference

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /out/inference /app/inference
EXPOSE 8081
CMD ["/app/inference"]
