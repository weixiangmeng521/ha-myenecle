ARG BUILD_FROM
FROM $BUILD_FROM

# Install requirements for add-on
RUN apk add --no-cache \
    bash \
    curl \
    perl \
    perl-html-entities \
    perl-html-parser

        
LABEL \
    io.hass.version="VERSION" \
    io.hass.type="addon" \
    io.hass.arch="armhf|aarch64|i386|amd64"

# Copy data for add-on
COPY run.sh /
RUN chmod a+x /run.sh

# 每半小时运行一次 run.sh
CMD [ "sh", "-c", "while true; do /run.sh; sleep 1800; done"]