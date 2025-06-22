# syntax=docker/dockerfile:1.4
FROM --platform=${TARGETPLATFORM} golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
ARG TARGETOS
ARG TARGETOSARCH
ARG SERVER_PORT
ARG LOGGING_LEVEL
ARG VERSION=dev
RUN apk add --no-cache make git
RUN make build TARGETOS=$TARGETOS TARGETOSARCH=$TARGETOSARCH VERSION=$VERSION

# Final stage
FROM --platform=${TARGETPLATFORM} gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=builder /app/k8s-controller .
ENV PORT=$SERVER_PORT
ENV LOGGING_LEVEL=$LOGGING_LEVEL
EXPOSE $SERVER_PORT
ENTRYPOINT ["/k8s-controller"] 