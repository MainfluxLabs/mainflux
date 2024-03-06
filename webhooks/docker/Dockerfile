FROM golang:1.18-alpine AS builder

WORKDIR /go/src/github.com/MainfluxLabs/mainflux
COPY ../../go.mod  ./
COPY ../../go.sum  ./

RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOARCH=amd64 \
  go build -mod=vendor -ldflags "-s -w" -o build/mainfluxlabs-webhooks cmd/webhooks/main.go \
  && mv build/mainfluxlabs-webhooks /exe

FROM scratch
COPY --from=builder /exe /
ENTRYPOINT ["/exe"]
