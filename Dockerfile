# syntax=docker/dockerfile:1

# ---- Build stage: compile a static Go binary ----
FROM golang:1.27-alpine AS build
WORKDIR /src

# Download dependencies first so this layer is cached between builds.
COPY go.mod ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api .

# ---- Runtime stage: minimal image, no shell, non-root user ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]
