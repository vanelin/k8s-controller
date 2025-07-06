# syntax=docker/dockerfile:1.7
FROM --platform=${TARGETPLATFORM} golang:1.24-alpine AS builder
WORKDIR /app
RUN go env -w GOMODCACHE=/root/.cache/go-build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build go mod download
COPY . .
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -v -o k8s-controller -ldflags "-X github.com/vanelin/k8s-controller.git/cmd.appVersion=$VERSION" main.go

# Final stage
FROM --platform=${TARGETPLATFORM} gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=builder /app/k8s-controller .

ARG SERVER_PORT=8080
ARG METRIC_PORT=8081
ARG LOGGING_LEVEL=debug
ARG KUBECONFIG
ARG NAMESPACE=default
ARG IN_CLUSTER=false
ARG ENABLE_LEADER_ELECTION=true
ARG LEADER_ELECTION_NAMESPACE=default

ENV PORT=$SERVER_PORT
ENV METRIC_PORT=$METRIC_PORT
ENV KUBECONFIG=$KUBECONFIG
ENV IN_CLUSTER=$IN_CLUSTER
ENV NAMESPACE=$NAMESPACE
ENV LOGGING_LEVEL=$LOGGING_LEVEL
ENV ENABLE_LEADER_ELECTION=$ENABLE_LEADER_ELECTION
ENV LEADER_ELECTION_NAMESPACE=$LEADER_ELECTION_NAMESPACE

LABEL org.opencontainers.image.source=https://github.com/vanelin/k8s-controller

EXPOSE 8080 8081
ENTRYPOINT ["/k8s-controller"]