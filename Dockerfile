# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.27.3-alpine3.24 AS builder

WORKDIR /app
RUN wget https://github.com/swaggo/swag/releases/download/v1.16.4/swag_1.16.4_Linux_x86_64.tar.gz && \
    tar -xzf swag_1.16.4_Linux_x86_64.tar.gz && \
    mv swag /usr/local/bin/swag && \
    rm swag_1.16.4_Linux_x86_64.tar.gz

COPY go.mod go.sum ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go install github.com/pressly/goose/v3/cmd/goose@latest

COPY . .
# Generate Swagger documentation
RUN ./generate-spec.sh

RUN CGO_ENABLED=0 GOOS=linux go build -o server

# Final stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/docs ./docs

EXPOSE 8000

ENV GIN_MODE=release
CMD ["sh", "-c", "goose -dir migrations postgres \"host=$DATABASE_HOST port=$DATABASE_PORT user=$POSTGRES_USER password=$POSTGRES_PASSWORD dbname=$DATABASE_NAME sslmode=disable search_path=bpl2,public\" up && ./server"]