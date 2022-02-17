ARG GO_VERSION=1.17.7

FROM golang:${GO_VERSION}-alpine AS build
RUN apk add --no-cache git
RUN apk --no-cache add ca-certificates

RUN addgroup -S updater && \
    adduser -S -u 1000 -g updater updater

WORKDIR /src
COPY ./go.mod ./go.sum ./
RUN go mod download

COPY ./ ./

RUN CGO_ENABLED=0 go mod tidy -compat=1.17

RUN CGO_ENABLED=0 go test -timeout 30s ./...

RUN CGO_ENABLED=0 go build \
    -installsuffix 'static' \
    -o /yaml-updater ./cmd/yaml-updater

FROM scratch AS final
COPY --from=build /yaml-updater /yaml-updater

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=build /etc/passwd /etc/passwd

USER updater

ENTRYPOINT ["/yaml-updater"]
