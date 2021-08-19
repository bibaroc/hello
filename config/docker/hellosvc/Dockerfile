FROM golang:1.17-alpine AS build

WORKDIR /src

ENV CGO_ENABLED 0
ENV GO111MODULE on

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build \
    -trimpath \
    -gcflags='-e -l' \
    -ldflags="-w -s -extldflags '-static'" \
    -o bin/hellosvc \
    ./cmd/hellosvc

FROM scratch AS prod
COPY --from=build /src/bin/hellosvc /bin/hellosvc
ENTRYPOINT [ "hellosvc" ]
