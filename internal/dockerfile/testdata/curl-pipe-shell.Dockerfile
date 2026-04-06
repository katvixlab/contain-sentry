FROM alpine
RUN curl -fsSL https://example.com/install.sh | sh
USER 1001
HEALTHCHECK CMD true
