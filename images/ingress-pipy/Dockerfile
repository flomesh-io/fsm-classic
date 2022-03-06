FROM flomesh/pipy-pjs:0.4.0-334

WORKDIR /ingress
COPY config/ingress-pipy/start.sh .

RUN apk --no-cache add jq

ENTRYPOINT ["/ingress/start.sh"]