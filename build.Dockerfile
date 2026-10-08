FROM ghcr.io/formancehq/base:22.04
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/orchestration /usr/bin/orchestration
ENV OTEL_SERVICE_NAME orchestration
ENTRYPOINT ["/usr/bin/orchestration"]
CMD ["serve"]
