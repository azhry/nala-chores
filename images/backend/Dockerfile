FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive
ENV OPENCODE_DISABLE_AUTOUPDATE=1

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    bash \
    ca-certificates \
    curl \
    git \
    jq \
    openssh-client \
    python3 \
    python3-yaml \
    tar \
  && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://opencode.ai/install | bash \
  && if ! command -v opencode >/dev/null 2>&1 && [ -x /root/.opencode/bin/opencode ]; then cp /root/.opencode/bin/opencode /usr/local/bin/opencode && chmod +x /usr/local/bin/opencode; fi \
  && command -v opencode

RUN useradd --create-home --shell /bin/bash runner \
  && mkdir -p /opt/runner/lib /opt/runner/schemas /workspace \
  && chown -R runner:runner /opt/runner /workspace

COPY opt/runner/ /opt/runner/
RUN chmod +x /opt/runner/entrypoint.sh /opt/runner/lib/*.sh

USER runner
WORKDIR /workspace
ENTRYPOINT ["/opt/runner/entrypoint.sh"]
