ARG BUILD_FROM
FROM $BUILD_FROM

# Install requirements for add-on
RUN apk add --no-cache 
        
LABEL \
    io.hass.version="VERSION" \
    io.hass.type="addon" \
    io.hass.arch="armhf|aarch64|i386|amd64"

# 拷贝二进制和启动脚本
COPY build/enecle-linux-amd64 /enecle-linux-amd64
COPY run.sh /run.sh

RUN chmod +x /enecle-linux-amd64 /run.sh

# 设置启动命令
CMD [ "/run.sh" ]