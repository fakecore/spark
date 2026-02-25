FROM nginx:alpine
RUN echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.20/main/" > /etc/apk/repositories

RUN apk update \
        && apk upgrade \
        && apk add --no-cache bash \
        bash-doc \
        bash-completion \
        && rm -rf /var/cache/apk/*

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

COPY dist/frontend /usr/share/nginx/html
# 只复制 Nginx 配置
COPY docker/frontend/nginx/site.conf /etc/nginx/conf.d/default.conf

# 静态文件将在运行时通过卷挂载

EXPOSE 9001

CMD ["nginx", "-g", "daemon off;"]
