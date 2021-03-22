FROM golang:1.16 as builder
ARG browser_ref
ARG browser_sha
ENV BUILD_DIR /tmp/browser

ADD . ${BUILD_DIR}
WORKDIR ${BUILD_DIR}

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X 'main.version=${browser_ref}' -X 'main.commit=${browser_sha}'" -o browser cmd/browser/main.go

FROM alpine:latest
RUN apk add --no-cache iputils ca-certificates net-snmp-tools procps &&\
    update-ca-certificates
COPY --from=builder /tmp/browser/browser /usr/bin/browser
EXPOSE 8888
CMD ["browser"]
