FROM digitalwonderland/base

RUN curl -Lo /usr/local/bin/docker-logstash-forwarder https://github.com/digital-wonderland/docker-logstash-forwarder/releases/download/latest/linux_amd64_docker-logstash-forwarder \
 && curl -Lo /usr/local/bin/logstash-forwarder https://github.com/digital-wonderland/docker-logstash-forwarder/releases/download/latest/linux_amd64_logstash-forwarder \
 && chmod 0755 /usr/local/bin/{docker-logstash-forwarder,logstash-forwarder}

ENTRYPOINT ["/usr/local/bin/docker-logstash-forwarder"]
