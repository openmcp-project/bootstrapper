# Stage 1: Download and extract ocm binary
FROM alpine AS downloader
ARG TARGETOS
ARG TARGETARCH
RUN apk add --no-cache curl tar
WORKDIR /tmp
# renovate: datasource=github-releases depName=ocm packageName=open-component-model/ocm
ARG OCM_VERSION=0.27.0
RUN curl -L -o ocm.tar.gz https://github.com/open-component-model/ocm/releases/download/v$OCM_VERSION/ocm-$OCM_VERSION-$TARGETOS-$TARGETARCH.tar.gz \
    && tar -xzf ocm.tar.gz

# Use distroless as minimal base image to package the component binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static-debian12:nonroot@sha256:627d6c5a23ad24e6bdff827f16c7b60e0289029b0c79e9f7ccd54ae3279fb45f
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
