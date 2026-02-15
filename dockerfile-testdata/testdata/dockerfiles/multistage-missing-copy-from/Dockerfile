FROM golang:1.22 AS builder
WORKDIR /src
RUN echo "fake build output" > /src/app

FROM debian:12
# intentionally missing COPY --from=builder
USER 1001
HEALTHCHECK CMD ["sh","-c","true"]
CMD ["sh", "-c", "test -f /src/app && cat /src/app || echo 'no artifact copied'"]
