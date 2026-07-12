FROM node:22 AS frontend-build
WORKDIR /ui
RUN corepack enable npm
COPY ui/package*.json ./
RUN npm ci
COPY ui/ ./
RUN npm test -- --watch=false && \
    npm run build -- --configuration production --output-path=dist/v2

FROM golang:1.26 AS genconplanner-base

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY ./internal ./internal
COPY ./cmd ./cmd
COPY ./static ./static

RUN go test ./...




# --------------------------
FROM genconplanner-base AS update

RUN groupadd -r appgroup && \
    useradd -r -g appgroup appuser && \
    go build -o /usr/local/bin/update ./cmd/update
COPY --chown=appuser:appgroup ./data ./data

RUN chown -R appuser:appgroup /usr/src/app
USER appuser

CMD ["/usr/local/bin/update", "-overrideDNS=true"]


# --------------------------
FROM genconplanner-base AS web

RUN groupadd -r appgroup && useradd -r -g appgroup appuser

COPY --chown=appuser:appgroup ./templates ./templates
COPY --chown=appuser:appgroup ./static ./static
COPY --chown=appuser:appgroup --from=frontend-build /ui/dist/v2/browser ./static/v2
RUN go build -o /usr/local/bin/web ./cmd/web && \
    chown -R appuser:appgroup /usr/src/app
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/ || exit 1

CMD ["/usr/local/bin/web"]
