FROM golang:1.12.12 as builder

ENV BUILD_DIR /tmp/browser

ADD . ${BUILD_DIR}
WORKDIR ${BUILD_DIR}

RUN go generate gitlab.inf.unibz.it/lter/browser/static
RUN CGO_ENABLED=0 GOOS=linux go build -o browser cmd/browser/main.go

FROM alpine:latest
RUN apk add --no-cache iputils ca-certificates net-snmp-tools procps &&\
    update-ca-certificates
COPY --from=builder /tmp/browser/browser /usr/bin/browser
EXPOSE 8888
CMD ["browser"]
