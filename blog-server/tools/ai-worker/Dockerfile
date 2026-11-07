FROM golang:1.25.3-bookworm AS go
FROM node:22-bookworm-slim
COPY --from=go /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}" GOMAXPROCS=1 GOCACHE=/opt/go-cache GOPATH=/opt/go GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local
RUN apt-get update && apt-get install -y --no-install-recommends python3 ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /opt/deps
COPY blog-web/package.json blog-web/package-lock.json ./
RUN npm ci --ignore-scripts && chmod -R a+rX /opt/deps
COPY blog-server/go.mod blog-server/go.sum /opt/server/
RUN cd /opt/server && GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org go mod download && chmod -R a+rX /opt/go
COPY blog-server/ /opt/server/
RUN cd /opt/server && go test -run '^$' -p 1 ./... && chown -R 10001:10001 /opt/go-cache
COPY blog-server/tools/ai-worker/check.py /opt/check.py
USER 10001:10001
WORKDIR /workspace
