# Stage 1: Download and extract ocm binary
FROM alpine@sha256:865b95f46d98cf867a156fe4a135ad3fe50d2056aa3f25ed31662dff6da4eb62 AS downloader
ARG TARGETOS
ARG TARGETARCH
RUN apk add --no-cache curl tar
WORKDIR /tmp
# renovate: datasource=github-releases depName=ocm packageName=open-component-model/ocm
ARG OCM_VERSION=0.34.3
RUN curl -L -o ocm.tar.gz https://github.com/open-component-model/ocm/releases/download/v$OCM_VERSION/ocm-$OCM_VERSION-$TARGETOS-$TARGETARCH.tar.gz \
    && tar -xzf ocm.tar.gz

# Use distroless as minimal base image to package the component binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static-debian12:nonroot@sha256:cba10d7abd3e203428e86f5b2d7fd5eb7d8987c387864ae4996cf97191b33764
ARG TARGETOS
ARG TARGETARCH
ARG COMPONENT
WORKDIR /
COPY bin/$COMPONENT.$TARGETOS-$TARGETARCH /<component>
# Copy ocm binary from downloader stage (adjust path if needed)
COPY --from=downloader /tmp/ocm /usr/local/bin/ocm
USER 65532:65532

# docker doesn't substitue args in ENTRYPOINT, so we replace this during the build script
ENTRYPOINT ["/<component>"]
