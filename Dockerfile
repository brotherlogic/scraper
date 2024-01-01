# syntax=docker/dockerfile:1

FROM golang:1.19 AS build

WORKDIR $GOPATH/src/github.com/brotherlogic/scraper

COPY go.mod ./
COPY go.sum ./

RUN mkdir proto
COPY proto/*.go ./proto/

RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 go build -o /scraper

##
## Deploy
##
FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=build /scraper /scraper

EXPOSE 8080
EXPOSE 8081

USER nonroot:nonroot

ENTRYPOINT ["/scraper"]