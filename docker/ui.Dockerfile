# Stage 1: Tailwind Build
FROM node:20-alpine AS tailwind
WORKDIR /build
COPY . .
RUN cd web/tailwind && npm install && npm run build

# Stage 2: Go Build
FROM golang:1.23-alpine AS build
WORKDIR /build
RUN apk add --no-cache git
COPY . .
RUN go install github.com/a-h/templ/cmd/templ@latest
RUN templ generate
RUN go mod tidy && go mod download
# Kopiere das generierte CSS aus der tailwind-stage
COPY --from=tailwind /build/web/static/tailwind.css web/static/tailwind.css
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ajnx-ui ./cmd/ui

# Stage 3: Final
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /ajnx-ui /ajnx-ui
COPY --from=build /build/web/static /web/static
EXPOSE 9858
USER nonroot
ENTRYPOINT ["/ajnx-ui"]
