FROM alpine
RUN apk add build-base
USER 1001
HEALTHCHECK CMD true
