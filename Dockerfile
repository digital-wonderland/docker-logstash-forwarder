FROM digitalwonderland/base

ENV GOPATH /var/lib/golang
ENV PATH $GOPATH/bin:$PATH

RUN mkdir $GOPATH

RUN yum install -y epel-release \
 && yum install -y hg git golang && yum clean all

RUN go get github.com/elasticsearch/logstash-forwarder \
 && go get github.com/tools/godep \
 && godep get github.com/digital-wonderland/docker-logstash-forwarder

ENTRYPOINT ["/var/lib/golang/bin/docker-logstash-forwarder"]
