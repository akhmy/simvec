FROM golang:1.26.1 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o core ./cmd/core

FROM alpine:latest

RUN apk add --no-cache ca-certificates curl && \
    curl -sSL "https://gu-st.ru/content/lending/russian_trusted_root_ca_pem.crt" \
    -o /usr/local/share/ca-certificates/russian_trusted_root_ca.crt && \
    curl -sSL "https://gu-st.ru/content/lending/russian_trusted_sub_ca_pem.crt" \
    -o /usr/local/share/ca-certificates/russian_trusted_sub_ca.crt && \
    update-ca-certificates

WORKDIR /app

COPY --from=builder /app/core .

EXPOSE 8080

CMD ["./core"]
