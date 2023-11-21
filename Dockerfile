FROM node:16 as NODE_BUILD
WORKDIR /go/src/github.com/siyuan-community/siyuan/
ADD . /go/src/github.com/siyuan-community/siyuan/
RUN cd app && npm install -g pnpm && pnpm install && pnpm run build

FROM golang:alpine as GO_BUILD
WORKDIR /go/src/github.com/siyuan-community/siyuan/
COPY --from=NODE_BUILD /go/src/github.com/siyuan-community/siyuan/ /go/src/github.com/siyuan-community/siyuan/
ENV GO111MODULE=on
ENV CGO_ENABLED=1
RUN apk add --no-cache gcc musl-dev git && \
    cd kernel && go build --tags fts5 -v -ldflags "-s -w -X github.com/siyuan-community/siyuan/kernel/util.Mode=prod" && \
    mkdir /opt/siyuan/ && \
    mv /go/src/github.com/siyuan-community/siyuan/app/appearance/ /opt/siyuan/ && \
    mv /go/src/github.com/siyuan-community/siyuan/app/stage/ /opt/siyuan/ && \
    mv /go/src/github.com/siyuan-community/siyuan/app/guide/ /opt/siyuan/ && \
    mv /go/src/github.com/siyuan-community/siyuan/app/changelogs/ /opt/siyuan/ && \
    mv /go/src/github.com/siyuan-community/siyuan/kernel/kernel /opt/siyuan/ && \
    find /opt/siyuan/ -name .git | xargs rm -rf

FROM alpine:latest
LABEL maintainer="Liang Ding<845765@qq.com>"

WORKDIR /opt/siyuan/
COPY --from=GO_BUILD /opt/siyuan/ /opt/siyuan/
RUN addgroup --gid 1000 siyuan && adduser --uid 1000 --ingroup siyuan --disabled-password siyuan && apk add --no-cache ca-certificates tzdata && chown -R siyuan:siyuan /opt/siyuan/

ENV TZ=Asia/Shanghai
ENV RUN_IN_CONTAINER=true
EXPOSE 6806

USER siyuan
ENTRYPOINT ["/opt/siyuan/kernel"]
