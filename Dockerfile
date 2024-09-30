FROM node:21 AS NODE_BUILD

WORKDIR /go/src/github.com/siyuan-community/siyuan/
ADD . /go/src/github.com/siyuan-community/siyuan/
RUN apt-get update && \
    apt-get install -y jq
RUN cd app && \
packageManager=$(jq -r '.packageManager' package.json) && \
if [ -n "$packageManager" ]; then \
    npm install -g $packageManager; \
else \
    echo "No packageManager field found in package.json"; \
    npm install -g pnpm; \
fi && \
pnpm install --registry=http://registry.npmjs.org/ --silent && \
pnpm run build
RUN apt-get purge -y jq
RUN apt-get autoremove -y
RUN rm -rf /var/lib/apt/lists/*

FROM golang:alpine AS GO_BUILD
WORKDIR /go/src/github.com/siyuan-community/siyuan/
COPY --from=NODE_BUILD /go/src/github.com/siyuan-community/siyuan/ /go/src/github.com/siyuan-community/siyuan/
ENV GO111MODULE=on
ENV CGO_ENABLED=1
RUN apk add --no-cache gcc musl-dev && \
    cd kernel && go build --tags fts5 -v -ldflags "-s -w" && \
    mkdir /opt/siyuan/ && \
    mv /go/src/github.com/siyuan-community/siyuan/app/appearance/ /opt/siyuan/ && \
    mv /go/src/github.com/siyuan-community/siyuan/app/stage/ /opt/siyuan/ && \
    mv /go/src/github.com/siyuan-community/siyuan/app/guide/ /opt/siyuan/ && \
    mv /go/src/github.com/siyuan-community/siyuan/app/changelogs/ /opt/siyuan/ && \
    mv /go/src/github.com/siyuan-community/siyuan/kernel/kernel /opt/siyuan/ && \
    mv /go/src/github.com/siyuan-community/siyuan/kernel/entrypoint.sh /opt/siyuan/entrypoint.sh && \
    find /opt/siyuan/ -name .git | xargs rm -rf

FROM alpine:latest
LABEL maintainer="Liang Ding<845765@qq.com>"

WORKDIR /opt/siyuan/
COPY --from=GO_BUILD /opt/siyuan/ /opt/siyuan/

RUN apk add --no-cache ca-certificates tzdata su-exec && \
    chmod +x /opt/siyuan/entrypoint.sh

ENV TZ=Asia/Shanghai
ENV HOME=/home/siyuan
ENV RUN_IN_CONTAINER=true
EXPOSE 6806

ENTRYPOINT ["/opt/siyuan/entrypoint.sh"]
CMD ["/opt/siyuan/kernel"]
