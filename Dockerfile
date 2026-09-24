# Stage 1: Download and extract ocm binary
FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS downloader
ARG TARGETOS
ARG TARGETARCH
RUN apk add --no-cache curl tar
WORKDIR /tmp
# renovate: datasource=github-releases depName=ocm packageName=open-component-model/ocm
ARG OCM_VERSION=0.50.0
RUN curl -L -o ocm.tar.gz https://github.com/open-component-model/ocm/releases/download/v$OCM_VERSION/ocm-$OCM_VERSION-$TARGETOS-$TARGETARCH.tar.gz \
    && tar -xzf ocm.tar.gz


FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS base
ARG TARGETOS
ARG TARGETARCH
ARG COMPONENT
RUN apk add --no-cache curl unzip git bash gettext jq yq kubectl
WORKDIR /
COPY bin/$COMPONENT.$TARGETOS-fips-$TARGETARCH /<component>
# Copy ocm binary from downloader stage (adjust path if needed)
COPY --from=downloader /tmp/ocm /usr/local/bin/ocm
USER 65532:65532

# docker doesn't substitue args in ENTRYPOINT, so we replace this during the build script
ENTRYPOINT ["/<component>"]
