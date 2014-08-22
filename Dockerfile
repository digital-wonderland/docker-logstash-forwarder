FROM digitalwonderland/base:centos6

ENV GOPATH /var/lib/golang
ENV PATH $GOPATH/bin:$PATH

RUN mkdir $GOPATH

RUN rpm -ivh https://dl.fedoraproject.org/pub/epel/6/x86_64/epel-release-6-8.noarch.rpm \
 && yum install -y hg git golang && yum clean all

RUN go get github.com/elasticsearch/logstash-forwarder \
 && go get github.com/tools/godep \
 && godep get github.com/dryewo/docker-logstash-forwarder

ENTRYPOINT ["/var/lib/golang/bin/docker-logstash-forwarder"]
