FROM alpine:latest
RUN echo "ok"
USER 1001
HEALTHCHECK CMD true
