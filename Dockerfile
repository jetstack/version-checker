FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

COPY . /app/
WORKDIR /app/

RUN go mod download -x

ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o ./bin/version-checker ./cmd/.


FROM alpine:3.21.3
LABEL description="Kubernetes utility for exposing used image versions compared to the latest version, as metrics."

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/bin/version-checker /usr/bin/version-checker

ENTRYPOINT ["/usr/bin/version-checker"]
