# Stage 1: Download the tools that are not packaged for the runtime image
FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf AS downloader
ARG TARGETOS
ARG TARGETARCH
RUN apk add --no-cache curl tar
WORKDIR /tmp
# renovate: datasource=github-releases depName=ocm packageName=open-component-model/ocm
ARG OCM_VERSION=0.55.0
RUN curl -fsSL -o ocm.tar.gz https://github.com/open-component-model/ocm/releases/download/v$OCM_VERSION/ocm-$OCM_VERSION-$TARGETOS-$TARGETARCH.tar.gz \
    && tar -xzf ocm.tar.gz
# renovate: datasource=github-releases depName=yq packageName=mikefarah/yq
ARG YQ_VERSION=v4.54.3
RUN curl -fsSL -o yq https://github.com/mikefarah/yq/releases/download/$YQ_VERSION/yq_${TARGETOS}_${TARGETARCH} \
    && chmod +x yq
# renovate: datasource=github-releases depName=kubectl packageName=kubernetes/kubernetes
ARG KUBECTL_VERSION=v1.37.2
RUN curl -fsSL -o kubectl https://dl.k8s.io/release/$KUBECTL_VERSION/bin/$TARGETOS/$TARGETARCH/kubectl \
    && chmod +x kubectl


# The FIPS base keeps the runtime OS FIPS-configured, as the build verifies.
FROM ghcr.io/gardenlinux/gardenlinux/fips:1877.28@sha256:60ce1d9bb98a00591e3cbaf962d370cb73c8e1edb0a39acde5c77ebbb23538d4 AS base
ARG TARGETOS
ARG TARGETARCH
ARG COMPONENT
USER 0
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl unzip git gettext-base jq \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /
COPY bin/$COMPONENT.$TARGETOS-fips-$TARGETARCH /<component>
COPY --from=downloader /tmp/ocm /tmp/yq /tmp/kubectl /usr/local/bin/
USER 65532:65532

# docker doesn't substitue args in ENTRYPOINT, so we replace this during the build script
ENTRYPOINT ["/<component>"]
