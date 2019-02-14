ARG GOVER=1.11.5
FROM golang:${GOVER} as build

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build

FROM alpine:3.8
RUN apk --no-cache add ca-certificates
COPY --from=build /build/mirror /usr/local/bin/
CMD mirror
