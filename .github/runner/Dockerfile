# A self-hosted GitHub Actions runner for the cardmarket-market label:
# Magic, Pokemon and YuGiOh's Market workflows, the three catalogs that
# don't fit ubuntu-latest's ~5-6h practical job ceiling even filtered.
# See README.md for what this actually deploys as and why.
FROM ubuntu:24.04

# https://github.com/actions/runner/releases - bump deliberately, this is
# not resolved at build time.
ARG RUNNER_VERSION=2.337.0

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        curl \
        ca-certificates \
        git \
        jq \
        tar \
        openssl \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m runner
WORKDIR /home/runner

# actions/setup-go (a step in run-bantool.yml) installs Go itself, pinned
# to go.mod, so it is not baked in here - one source of truth for the
# version instead of two to keep in sync.
RUN curl -fsSL -o actions-runner.tar.gz \
        "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz" \
    && tar xzf actions-runner.tar.gz \
    && rm actions-runner.tar.gz \
    && ./bin/installdependencies.sh \
    && chown -R runner:runner /home/runner

COPY entrypoint.sh /home/runner/entrypoint.sh
RUN chmod +x /home/runner/entrypoint.sh

USER runner
ENTRYPOINT ["/home/runner/entrypoint.sh"]
