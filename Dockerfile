FROM golang:1.22 AS genconplanner-base

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY ./internal ./internal
COPY ./cmd ./cmd



# --------------------------
FROM genconplanner-base AS update

RUN go build -o /usr/local/bin/update ./cmd/update
COPY ./data ./data
CMD ["/bin/sh", "-c", "/usr/local/bin/update -db=postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@db:5432/$POSTGRES_DB?sslmode=disable -overrideDNS=true"]
# CMD ["/bin/sh", "-c", "/usr/local/bin/update -db=postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@db:5432/$POSTGRES_DB?sslmode=disable -eventFile=./data/debug_events.xlsx"]


# --------------------------
FROM genconplanner-base AS web

COPY ./templates ./templates
COPY ./static ./static
RUN go build -o /usr/local/bin/web ./cmd/web

EXPOSE 8080

CMD ["/bin/sh", "-c", "/usr/local/bin/web -port=8080 -db=postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@db:5432/$POSTGRES_DB?sslmode=disable"]
