FROM debian:12
RUN curl -fsSLO https://example.com/app.tar.gz  && echo "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef  app.tar.gz" > /tmp/SHA256SUMS  && sha256sum -c /tmp/SHA256SUMS
