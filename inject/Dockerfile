FROM golang:1.16

WORKDIR /app/webhook

COPY out/webhook webhook

COPY spec spec

RUN chmod 755 webhook