# Stage 1: Download and extract ocm binary
FROM alpine@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS downloader
ARG TARGETOS
ARG TARGETARCH
RUN apk add --no-cache curl tar
WORKDIR /tmp
# renovate: datasource=github-releases depName=ocm packageName=open-component-model/ocm
ARG OCM_VERSION=0.40.0
RUN curl -L -o ocm.tar.gz https://github.com/open-component-model/ocm/releases/download/v$OCM_VERSION/ocm-$OCM_VERSION-$TARGETOS-$TARGETARCH.tar.gz \
    && tar -xzf ocm.tar.gz


FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS base
ARG TARGETOS
ARG TARGETARCH
ARG COMPONENT
RUN apk add --no-cache curl unzip git bash gettext jq yq kubectl
WORKDIR /
COPY bin/$COMPONENT.$TARGETOS-$TARGETARCH /<component>
# Copy ocm binary from downloader stage (adjust path if needed)
COPY --from=downloader /tmp/ocm /usr/local/bin/ocm
USER 65532:65532

# docker doesn't substitue args in ENTRYPOINT, so we replace this during the build script
ENTRYPOINT ["/<component>"]
