FROM debian:12

COPY .env /app/.env
COPY id_rsa /root/.ssh/id_rsa

ADD kubeconfig /root/.kube/config
ADD https://example.com/file.tar.gz /tmp/file.tar.gz
