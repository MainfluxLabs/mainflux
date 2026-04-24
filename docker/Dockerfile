FROM golang:1.25.8-alpine AS builder
ARG SVC
ARG VERSION
ARG COMMIT
ARG TIME

WORKDIR /go/src/github.com/MainfluxLabs/mainflux
COPY . .
RUN apk update \
    && apk add make\
    && make $SVC \
    && mv build/mainfluxlabs-$SVC /exe \
    && mkdir -p /etc/ssl/certs/crl \
    && chown -R 65532:65532 /etc/ssl/certs/crl

# scratch has no user concept; distroless/static provides a non-root user required by security policy.
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /exe /
COPY --from=builder /etc/ssl/certs/crl /etc/ssl/certs/crl
USER nonroot
ENTRYPOINT ["/exe"]
