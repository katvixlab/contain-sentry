FROM alpine
RUN echo "ok"
USER root
HEALTHCHECK CMD true
