FROM alpine:3.14

WORKDIR /repo
COPY config/repo-init/scripts/ .

RUN apk --no-cache add curl jq

ENTRYPOINT ["/repo/init.sh"]
