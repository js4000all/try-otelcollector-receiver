FROM golang:latest
LABEL version="0.1.0" maintainer="js4000all@gmail.com"

ARG OCB_VERSION=0.158.0

RUN set -x \
  && apt-get update \
  && apt-get upgrade -y \
  && apt-get install -y --no-install-recommends \
    git \
    less \
    vim \
    curl \
    ca-certificates \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL \
      "https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/cmd%2Fbuilder%2Fv${OCB_VERSION}/ocb_${OCB_VERSION}_linux_amd64" \
      -o /usr/local/bin/ocb \
    && chmod +x /usr/local/bin/ocb
RUN ocb version

RUN useradd -m -s /bin/bash dev

COPY ./profile.d/alias.sh /etc/profile.d

ENV LESSCHARSET=utf-8
ENV PAGER=less

WORKDIR /work
USER dev
