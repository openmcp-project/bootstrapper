# renovate: datasource=docker depName=ghcr.io/open-component-model/cli
ARG OCM_VERSION=0.17.0

FROM ghcr.io/open-component-model/cli:${OCM_VERSION} AS ocm-cli

FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS base
ARG TARGETOS
ARG TARGETARCH
ARG COMPONENT
RUN apk add --no-cache curl unzip git bash gettext jq yq kubectl
WORKDIR /
COPY bin/$COMPONENT.$TARGETOS-$TARGETARCH /<component>
COPY --from=ocm-cli /ocm /usr/local/bin/ocm
USER 65532:65532

# docker doesn't substitue args in ENTRYPOINT, so we replace this during the build script
ENTRYPOINT ["/<component>"]
