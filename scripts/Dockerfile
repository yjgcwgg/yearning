# ─────────────────────────────────────────
# Stage 1: 构建前端
# ─────────────────────────────────────────
FROM node:18-alpine AS frontend

WORKDIR /build
COPY gemini-next-next/package.json gemini-next-next/yarn.lock* ./
RUN npm install --legacy-peer-deps

COPY gemini-next-next/ ./
RUN npm run build

# ─────────────────────────────────────────
# Stage 2: 构建后端
# ─────────────────────────────────────────
FROM golang:1.22-alpine AS backend

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache git

ENV GOPROXY=https://goproxy.cn,direct
ENV GONOSUMCHECK=*
ENV GONOSUMDB=*

WORKDIR /build
COPY Yearning-next/go.mod Yearning-next/go.sum ./
RUN go mod download

COPY Yearning-next/ ./

# 嵌入前端产出
COPY --from=frontend /build/dist ./src/service/dist/

# chat 占位（go:embed 要求目录非空）
RUN mkdir -p src/service/chat/server/app && \
    echo '<!DOCTYPE html><html><body></body></html>' > src/service/chat/server/app/index.html

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /Yearning .

# ─────────────────────────────────────────
# Stage 3: 最终镜像
# ─────────────────────────────────────────
FROM alpine:3.20

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk update \
    && apk add --no-cache ca-certificates bash tzdata libc6-compat dumb-init \
    && cp -rf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

COPY --from=backend /Yearning /opt/Yearning
COPY Yearning-next/conf.toml.template /opt/conf.toml

WORKDIR /opt

EXPOSE 8000

ENV MYSQL_USER=""
ENV MYSQL_PASSWORD=""
ENV MYSQL_ADDR=""
ENV MYSQL_DB="yearning"
ENV SECRET_KEY="dbcjqheupqjsuwsm"
ENV IS_DOCKER="is_docker"

ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["bash", "-c", "/opt/Yearning run"]
