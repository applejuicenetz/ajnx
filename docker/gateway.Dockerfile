# Stage 1: Build
FROM golang:1.24-alpine AS build
WORKDIR /build
RUN apk add --no-cache git
COPY . .
# Templ installieren und generieren, damit go mod tidy die UI-Pakete (die im selben Modul liegen) nicht als externe Abhängigkeiten sucht
RUN go install github.com/a-h/templ/cmd/templ@latest
RUN templ generate
RUN go mod tidy && go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ajnx-gateway ./cmd/gateway

# Stage 2: Final
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /ajnx-gateway /ajnx-gateway
EXPOSE 9859
USER nonroot
ENTRYPOINT ["/ajnx-gateway"]
