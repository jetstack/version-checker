FROM alpine:3.12
LABEL description="Kubernetes utility for exposing used image versions compared to the latest version, as metrics."

RUN apk --no-cache add ca-certificates

COPY ./bin/version-checker-linux /usr/bin/version-checker

CMD ["/usr/bin/version-checker"]
