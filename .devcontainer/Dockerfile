FROM golang:latest
LABEL version="0.1.0" maintainer="js4000all@gmail.com"

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
RUN useradd -m -s /bin/bash dev

COPY ./profile.d/alias.sh /etc/profile.d

ENV LESSCHARSET=utf-8
ENV PAGER=less

WORKDIR /work
USER dev
