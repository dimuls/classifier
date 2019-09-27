FROM alpine:3 AS builder

RUN apk add --no-cache curl

RUN curl -L http://download.cdn.yandex.net/mystem/mystem-3.1-linux-64bit.tar.gz \
| tar xz



FROM ubuntu:18.04

COPY --from=builder /mystem /mystem
RUN chmod +x /mystem

ENV MYSTEM_BIN_PATH=/mystem

COPY classifier /classifier
RUN chmod +x /classifier

ENTRYPOINT ["/classifier"]