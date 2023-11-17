# That's the only place where you're supposed to specify version of Tunnel.
ARG TUNNEL_VERSION=0.46.1

FROM aquasec/trivy:${TUNNEL_VERSION}

# An ARG declared before a FROM is outside of a build stage, so it can't be used in any
# instruction after a FROM. To use the default value of an ARG declared before the first
# FROM use an ARG instruction without a value inside of a build stage.
ARG TUNNEL_VERSION

RUN adduser -u 10000 -D -g '' scanner scanner

COPY scanner-tunnel /home/scanner/bin/scanner-tunnel

ENV TUNNEL_VERSION=${TUNNEL_VERSION}

USER scanner

ENTRYPOINT ["/home/scanner/bin/scanner-tunnel"]
