FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/seed-demo ./cmd/seed-demo

FROM alpine:3.22
RUN addgroup -S zyra && adduser -S -G zyra zyra
WORKDIR /app
COPY --from=build /out/server /out/seed-demo ./
USER zyra
EXPOSE 8080
CMD ["/bin/sh", "-c", "/app/seed-demo && exec /app/server"]
