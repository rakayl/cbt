# ====== BUILD STAGE ======
FROM golang:1.23-alpine AS builder

WORKDIR /app

# install git (kadang dibutuhkan untuk dependency)
RUN apk add --no-cache git

# copy dependency file dulu (biar cache optimal)
COPY go.mod go.sum ./
RUN go mod download

# copy semua source
COPY . .

# build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o cbt-v2 ./cmd/api

# ====== FINAL STAGE ======
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# copy binary dari builder
COPY --from=builder /app/cbt-v2 .

EXPOSE 1102

CMD ["./cbt-v2"]