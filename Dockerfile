FROM node:22 AS frontend-build
WORKDIR /ui
COPY ui/package*.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY ui/ ./
RUN npm test -- --watch=false
RUN npm run build -- --configuration production --output-path=dist/v2

FROM golang:1.26 AS genconplanner-base

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY ./internal ./internal
COPY ./cmd ./cmd

RUN go test ./...



# --------------------------
FROM genconplanner-base AS update

RUN go build -o /usr/local/bin/update ./cmd/update
COPY ./data ./data
CMD ["/usr/local/bin/update", "-overrideDNS=true"]


# --------------------------
FROM genconplanner-base AS web

COPY ./templates ./templates
COPY ./static ./static
COPY --from=frontend-build /ui/dist/v2/browser ./static/v2
RUN go build -o /usr/local/bin/web ./cmd/web

EXPOSE 8080

CMD ["/usr/local/bin/web"]
