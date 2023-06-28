FROM alpine
COPY sonar-badge-proxy /bin/
ENTRYPOINT ["sonar-badge-proxy"]