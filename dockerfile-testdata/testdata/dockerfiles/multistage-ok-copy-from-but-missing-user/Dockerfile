FROM golang:1.22 AS builder
WORKDIR /src
RUN echo "hello" > /src/app

FROM debian:12
COPY --from=builder /src/app /app/app
HEALTHCHECK CMD ["sh","-c","test -f /app/app"]
CMD ["sh", "-c", "cat /app/app"]
