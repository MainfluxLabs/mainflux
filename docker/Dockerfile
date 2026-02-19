FROM golang:1.24.13-alpine AS builder
ARG SVC
ARG VERSION
ARG COMMIT
ARG TIME

WORKDIR /go/src/github.com/MainfluxLabs/mainflux
COPY . .
RUN apk update \
    && apk add make\
    && make $SVC \
    && mv build/mainfluxlabs-$SVC /exe

# scratch has no user concept; distroless/static provides a non-root user required by security policy.
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /exe /
USER nonroot
ENTRYPOINT ["/exe"]
