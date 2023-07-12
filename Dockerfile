FROM golang:1.20-alpine as builder

RUN apk --no-cache add make

COPY . /app/
WORKDIR /app/

RUN make build


FROM alpine:3.18.2
LABEL description="Kubernetes utility for exposing used image versions compared to the latest version, as metrics."

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/bin/version-checker-linux /usr/bin/version-checker

ENTRYPOINT ["/usr/bin/version-checker"]
