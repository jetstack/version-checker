FROM gcr.io/distroless/base-debian12:nonroot
LABEL description="Kubernetes utility for exposing used image versions compared to the latest version, as metrics."
ARG TARGETARCH
ARG TARGETOS
ADD dist/cmd-$TARGETOS-$TARGETARCH /usr/bin/version-checker
ENTRYPOINT ["/usr/bin/version-checker"]
