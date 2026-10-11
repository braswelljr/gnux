# syntax=docker/dockerfile:1

# ---- build stage ---------------------------------------------------------------
FROM golang:1.25-alpine AS build

WORKDIR /src

# Cache module downloads separately from the source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Version is injected at link time; override with --build-arg VERSION=<tag>.
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X github.com/braswelljr/gnux/internal/common.Version=${VERSION}" \
    -o /out/gnux .

# gnux picks the tool from the name it is invoked under, so one link per tool
# makes each of them (rmx, cpx, lsx, ...) a command of its own.
RUN cd /out && for t in $(./gnux --list); do ln -s gnux "$t"; done

# ---- runtime stage -------------------------------------------------------------
FROM alpine:3.20

# Run as an unprivileged user; mount the directory to operate on at /work.
RUN adduser -D -u 10001 gnux
COPY --from=build /out/ /usr/local/bin/

USER gnux
WORKDIR /work

ENTRYPOINT ["gnux"]
CMD ["--help"]
