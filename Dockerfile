FROM alpine:3.17.3
LABEL description="Kubernetes utility for exposing used image versions compared to the latest version, as metrics."

RUN apk --no-cache add ca-certificates

COPY ./bin/version-checker-linux /usr/bin/version-checker

ENTRYPOINT ["/usr/bin/version-checker"]
