###############################################################
# pinning image to Ubi8
###############################################################
FROM registry.access.redhat.com/ubi8-minimal


###############################################################
# dependencies
###############################################################
RUN microdnf update && microdnf install shadow-utils 

###############################################################
# add non privileged curl user
###############################################################
RUN groupadd curl_group && useradd -G curl_group curl_user

###############################################################
# set user
###############################################################
USER curl_user

###############################################################
# set entrypoint
###############################################################
WORKDIR /
COPY "dockerfiles/curl/entrypoint.sh" "entrypoint.sh"
COPY LICENSE /licenses/fsm-LICENSE

CMD ["curl"]
ENTRYPOINT ["/entrypoint.sh"]