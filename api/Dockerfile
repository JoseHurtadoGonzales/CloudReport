# ---------- build stage ----------
FROM golang:1.23-alpine AS build

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod ./
# go.sum is created on first build; tolerate its absence in CI by adding -mod=mod
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cloud-report ./cmd/server

# ---------- runtime stage ----------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /out/cloud-report /usr/local/bin/cloud-report

ENV HTTP_PORT=5488
EXPOSE 5488

ENTRYPOINT ["/usr/local/bin/cloud-report"]
