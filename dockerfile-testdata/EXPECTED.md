# Dockerfile test fixtures for ContainSentry

Each folder contains a `Dockerfile` and the expected rule IDs that should trigger.

Note: DF002 (`tag without digest`) is expected for most `FROM image:tag` cases.


## Matrix

- `copy-add-secrets`: DF002, DF010, DF011, DF014, DF015
- `digest-pinned-ok`: — (no findings expected)
- `final-missing-healthcheck`: DF002, DF022
- `final-missing-user`: DF002, DF005
- `latest-tag`: DF001, DF002
- `multistage-missing-copy-from`: DF002, DF003
- `multistage-ok-copy-from-but-missing-user`: DF002, DF005
- `run-apk-add-without-no-cache`: DF002, DF019
- `run-apt-cache-mount-without-sharing-locked`: DF002, DF021
- `run-apt-install-without-cleanup`: DF002, DF018
- `run-apt-update-separate`: DF002, DF016
- `run-apt-upgrade`: DF002, DF017
- `run-cache-mount-without-id`: DF002, DF020
- `run-curl-pipe-shell`: DF002, DF012, DF013
- `run-download-with-verification-ok`: DF002
- `secret-env-arg`: DF002, DF008, DF009
- `single-stage-with-build-tools`: DF002, DF004
- `tag-without-digest`: DF002
- `user-root`: DF002, DF006
- `user-uid-0`: DF002, DF006
- `user-uid-1000-fail`: DF002, DF007
- `user-uid-1001-ok`: DF002
