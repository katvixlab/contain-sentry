# intentionally insecure Dockerfile for negative tests

FROM ubuntu:latest

# 1) Секреты в ARG/ENV (запрещено)
ARG API_TOKEN=super-secret-token
ENV DB_PASSWORD=passw0rd

# 2) Опасный ADD (локальный) вместо COPY (не рекомендуется)
ADD ./app /app

# 3) Опасный ADD по URL без checksum (запрещено)
ADD https://example.com/tool.tar.gz /tmp/tool.tar.gz

# 4) apt-get update отдельным слоем (не рекомендуется/запрещено по политике)
RUN apt-get update

# 5) Установка пакетов без очистки /var/lib/apt/lists/* (нежелательно)
RUN apt-get install -y curl ca-certificates

# 6) Неуправляемая загрузка + curl | sh (запрещено)
RUN curl -fsSL https://example.com/install.sh | sh

# 7) Cache mount без id и без sharing=locked (для ключевых сборок запрещено)
RUN --mount=type=cache,target=/var/lib/apt \
    apt-get update && apt-get install -y git

# 8) Копирование “всего контекста” (опасно без строгого .dockerignore)
COPY . /src

# 9) Явно root (или отсутствие USER) -> запуск под root (запрещено)
USER root

CMD ["bash"]