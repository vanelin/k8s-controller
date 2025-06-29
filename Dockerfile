# syntax=docker/dockerfile:1.7
FROM --platform=${TARGETPLATFORM} golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -v -o k8s-controller -ldflags "-X github.com/vanelin/k8s-controller.git/cmd.appVersion=$VERSION" main.go

# Final stage
FROM --platform=${TARGETPLATFORM} gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=builder /app/k8s-controller .
ARG SERVER_PORT
ARG LOGGING_LEVEL
ARG KUBECONFIG
ENV PORT=$SERVER_PORT
ENV LOGGING_LEVEL=$LOGGING_LEVEL
ENV KUBECONFIG=$KUBECONFIG
LABEL org.opencontainers.image.source=https://github.com/vanelin/k8s-controller
EXPOSE $SERVER_PORT
ENTRYPOINT ["/k8s-controller"]