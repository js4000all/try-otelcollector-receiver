FROM golang:alpine
LABEL version="0.1.0" maintainer="js4000all@gmail.com"

RUN set -x \
  && apk update --no-cache \
  && apk upgrade --no-cache \
  && apk add --no-cache \
    git \
    less \
    vim \
    curl \
  && adduser --disabled-password dev

COPY ./profile.d/alias.sh /etc/profile.d

ENV LESSCHARSET=utf-8
ENV PAGER=less

WORKDIR /work
USER dev
