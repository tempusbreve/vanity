#
# builder
#

FROM golang:buster AS builder

ENV GO111MODULE=on \
    GOFLAGS="-tags=netgo -trimpath" \
    LDFLAGS="-s -w -extldflags -static" \
    CGO_ENABLED=0

COPY . /src/

WORKDIR /src/

RUN go build -i -ldflags "${LDFLAGS}" -o vanity .

#
# Final Image
#

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder --chown=1000:1000 /src/vanity /vanity

EXPOSE 39999

USER 1000:1000

ENTRYPOINT ["/vanity", "serve", "-b", "0.0.0.0:39999"]
