FROM golang:1.14 AS build-env
LABEL maintainer "Jimmy Zelinskie <jimmyzelinskie+git@gmail.com>"

# Install OS-level dependencies.
# RUN apk add --no-cache curl git

# Copy our source code into the container.
WORKDIR /go/src/github.com/chihaya/chihaya
COPY . /go/src/github.com/chihaya/chihaya

# Install our golang dependencies and compile our binary.
RUN CGO_ENABLED=0 go build
RUN CGO_ENABLED=0 go install -v ./...
CMD ["./chihaya", "--config", "config.yaml"]