# syntax=docker/dockerfile:1

# Two images come from this file. The default (the last stage) is Alpine with
# gnux on the PATH; `--target scratch` is gnux alone, a busybox-style base
# with nothing else in it. The Release workflow publishes both to GHCR.

# ---- build stage ---------------------------------------------------------------
FROM golang:1.26-alpine AS build

WORKDIR /src

# Cache module downloads separately from the source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Version is injected at link time; override with --build-arg VERSION=<tag>.
# timetzdata embeds the time zone database, which the scratch image has no
# copy of: datex and lsx -l show local times from TZ there too.
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -tags timetzdata \
    -ldflags "-s -w -X github.com/braswelljr/gnux/internal/common.Version=${VERSION}" \
    -o /out/bin/gnux .

# gnux picks the tool from the name it is invoked under, so one link per tool
# makes each of them (rmx, cpx, lsx, ...) a command of its own. The GNU names
# are linked too in the scratch image's tree, where nothing else owns them.
RUN cd /out/bin && for t in $(./gnux --list); do ln -s gnux "$t"; done
RUN mkdir -p /scratch/bin /scratch/tmp /scratch/work && chmod 1777 /scratch/tmp && \
    cp -a /out/bin/. /scratch/bin/ && \
    cd /scratch/bin && for t in $(./gnux --list); do ln -s gnux "${t%x}"; done && ln -s gnux '['

# ---- scratch: gnux and nothing else ------------------------------------------------
FROM scratch AS scratch

# COPY makes everything root's; only the working directory is the user's.
COPY --from=build /scratch/ /
COPY --from=build --chown=10001:10001 /scratch/work /work
ENV PATH=/bin
USER 10001:10001
WORKDIR /work

ENTRYPOINT ["gnux"]
CMD ["--help"]

# ---- runtime stage (the default) -------------------------------------------------
FROM alpine:3.20

# Run as an unprivileged user; mount the directory to operate on at /work.
RUN adduser -D -u 10001 gnux
COPY --from=build /out/bin/ /usr/local/bin/

USER gnux
WORKDIR /work

ENTRYPOINT ["gnux"]
CMD ["--help"]
