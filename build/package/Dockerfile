###########################################################
# Build stage
###########################################################
FROM golang:1.25-alpine@sha256:26111811bc967321e7b6f852e914d14bede324cd1accb7f81811929a6a57fea9 AS builder

WORKDIR /app

RUN apk add --no-cache \
    git \
    ca-certificates \
    ;

# Copy go mod files first for better caching
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN ./scripts/build.sh

###########################################################
# Runtime stage
###########################################################
FROM scratch

COPY --from=builder /app/bin/kubesnake /kubesnake

ENTRYPOINT ["/kubesnake"]
