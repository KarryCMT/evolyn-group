FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/
RUN CGO_ENABLED=0 go build -o /bin/evolyn-core ./cmd/api

FROM alpine
RUN adduser -D -u 10001 appuser
COPY --from=builder /bin/evolyn-core /evolyn-core
COPY config/app.example.yaml /config/app.yaml
USER appuser
EXPOSE 8080
ENTRYPOINT ["/evolyn-core", "--config", "/config/app.yaml"]
