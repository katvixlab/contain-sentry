FROM ubuntu:22.04
RUN apt-get update && apt-get install -y build-essential make gcc  && rm -rf /var/lib/apt/lists/*
HEALTHCHECK CMD ["bash","-lc","true"]
CMD ["bash", "-lc", "echo done"]
