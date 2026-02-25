FROM alpine:3.20
RUN echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.20/main/" > /etc/apk/repositories

RUN apk update \
        && apk upgrade \
        && apk add --no-cache bash file \
        bash-doc \
        bash-completion \
        && rm -rf /var/cache/apk/*

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 这里不再进行构建，而是在运行时通过卷挂载预构建的二进制文件
# 为了保持一致性，我们可以复制启动脚本
# Keep a default config in the image, but in production this is typically
# overridden via bind-mount or env overrides.
COPY docker/backend/config ./hack/config
COPY dist/backend/projecttemplate .

EXPOSE 8888

CMD ["./projecttemplate", "--conf", "/app/hack/config/config.yaml"]
