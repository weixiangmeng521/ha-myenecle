ARG BUILD_FROM
FROM $BUILD_FROM

# Install requirements for add-on
RUN \
    apk add --no-cache 

LABEL \
    io.hass.version="VERSION" \
    io.hass.type="addon" \
    io.hass.arch="armhf|aarch64|i386|amd64"

# Copy data for add-on
COPY run.sh /
RUN chmod a+x /run.sh

CMD [ "/run.sh" ]

