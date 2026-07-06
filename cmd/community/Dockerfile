FROM alpine:3.20.3
LABEL maintainer="SigInsight"
WORKDIR /root

ARG OS="linux"
ARG TARGETARCH

RUN apk update && \
    apk add --no-cache ca-certificates wget && \
    rm -rf /var/cache/apk/*

COPY ./target/${OS}-${TARGETARCH}/signoz-community /root/signoz
COPY ./templates/email /root/templates
COPY frontend/build/ /etc/signoz/web/

RUN chmod 755 /root /root/signoz

EXPOSE 8080

ENTRYPOINT ["./signoz", "server"]
