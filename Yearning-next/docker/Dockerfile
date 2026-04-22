ARG BASEIMAGES=3.20.2
FROM alpine:${BASEIMAGES} AS builder

LABEL maintainer="chaiyd <chaiyd.cn@gmail.com>,henryyee <henry@yearning.io>"

ARG YEARNING_VER=v3.1.5

ARG TARGETARCH

ARG YEARNING_URL=https://github.com/cookieY/Yearning/releases/download/${YEARNING_VER}/Yearning-${YEARNING_VER}-linux-${TARGETARCH}.zip

RUN wget -cO yearning.zip $YEARNING_URL && \
    unzip yearning.zip && \
    mv Yearning/* /opt


FROM  --platform=$TARGETPLATFORM alpine:${BASEIMAGES}

ARG TARGETPLATFORM

LABEL maintainer="chaiyd <chaiyd.cn@gmail.com>"

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk update  \
    && apk add --no-cache ca-certificates bash tree tzdata libc6-compat dumb-init \
    && cp -rf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

COPY --from=builder /opt/Yearning /opt/Yearning
COPY --from=builder /opt/conf.toml /opt/conf.toml

WORKDIR /opt/

EXPOSE 8000

#ENV Y_LANG  "zh_CN"
#ENV SECRET_KEY  "dbcjqheupqjsuwsm"
#ENV MYSQL_ADDR  ""
#ENV MYSQL_USER  ""
#ENV MYSQL_PASSWORD  ""
#ENV MYSQL_DB  ""

ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["bash", "-c", "/opt/Yearning run"]
