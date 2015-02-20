# Docker Logstash Integration
[![Build Status](https://travis-ci.org/digital-wonderland/docker-logstash-forwarder.svg)](https://travis-ci.org/digital-wonderland/docker-logstash-forwarder)
[![Go Walker](http://gowalker.org/api/v1/badge)](http://gowalker.org/github.com/digital-wonderland/docker-logstash-forwarder)
[![Gobuild Download](http://gobuild.io/badge/github.com/digital-wonderland/docker-logstash-forwarder/download.png)](http://gobuild.io/github.com/digital-wonderland/docker-logstash-forwarder)

Ship all your logs from [Docker](http://www.docker.io/) (including in container logs) to [Logstash](http://logstash.net/) via [logstash-forwarder](https://github.com/elasticsearch/logstash-forwarder) (aka lumberjack).

This means:

* any Docker log files
* whatever log files you configure logstash-forwarder to ship within a container (just put a config at ```/etc/logstash-forwarder.conf```, only the ```files``` section gets evaluated while ```network``` section is globally configured).

## Why?

I wasn't too happy with existing possibilities and while I know that the Docker team is working on a solution, this scratches my itch right now.

Also I didn't see an obvious way to extend [docker-gen](https://github.com/jwilder/docker-gen) to handle generic in container templates.

Besides that, how much reason do you need to play with Go & Docker? ;-)


## How it works:

```docker-logstash-forwarder``` listens to Docker events and continually restarts a logstash-forwarder instance, after refreshing its configuration, every ```laziness``` seconds after a new event was received (to avoid unnecessary restarts - configurable via ```-laziness``` flag - defaults to 5 seconds).

For every running container the docker log file is added and it is checked if a logstash-forwarder config exists within the container at ```/etc/logstash-forwarder.conf```.

If an in container specific config exists, the path of all files will be expanded to be valid within the logstash-forwarder container before adding them to the global configuration.

This requires the following (in container defaults in brackets):

* read-only access to the directory containing your docker data (```/var/lib/docker```)
* connection to Docker (```unix:///var/run/docker.sock```)
* connection to Logstash (```logstash:5043```)

### Read-only access to Docker data:

Mount the directory containing your Docker data into the containers ```/var/lib/docker``` - i.e. run the container with ```-v /var/lib/docker:/var/lib/docker:ro``` (assuming your Docker files are stored in ```/var/lib/docker``` on the host).

### Connection with Docker:

For communication with Docker the following endpoints are evaluated:

1. whatever is passed via the ```-docker``` command line flag
2. the ```$DOCKER_HOST``` environment variable
3.  ```unix:///var/run/docker.sock```

It is suggested to use the later - as in run the container with ```-v /var/run/docker.sock:/var/run/docker.sock```

Behind the screens [fsouza/go-dockerclient](https://github.com/fsouza/go-dockerclient/) is used for communication with Docker.

### Connection with Logstash:

For communication with Logstash the following endpoints are evaluated:

1. whatever is passed via the ```-logstash``` command line flag
2. the ```$LOGSTASH_HOST``` environment variable
3. ```logstash:5043```

This allows you to ```docker -link``` your [Logstash](https://github.com/digital-wonderland/docker-logstash) instance to the containers ```logstash``` host.

#### Certificate Handling:

logstash-forwarder authentication can be managed in the following ways:

1. specify a custom config pointing to some imported volume containing the required cert & key via the ```-config``` flag (only the ```network``` section is evaluated)
2. make your keys available bellow ```/mnt/logstash-forwarder```

## TL;DR / Quickstart:

If you have my [elasticsearch](https://registry.hub.docker.com/u/digitalwonderland/elasticsearch/) & [logstash](https://registry.hub.docker.com/u/digitalwonderland/logstash/) containers running just do

    $ docker pull digitalwonderland/logstash-forwarder
    $ docker run -d --name logstash-forwarder -v /var/lib/docker:/var/lib/docker:ro -v /var/run/docker.sock:/var/run/docker.sock --link logstash:logstash --volumes-from logstash digitalwonderland/logstash-forwarder

If you start from scratch / use [Vagrant](http://www.vagrantup.com/) / are on a Mac: just clone this repository and run ```vagrant up```. This gives you a VM based on [CoreOS](https://coreos.com/) (which is awesome btw) running those 3 containers & [Kibana](http://www.elasticsearch.org/overview/kibana/) listening to [localhost:5601](http://localhost:5601) (Docker listens to [localhost:2375](http://localhost:2375/containers/json)).

## Known Issues:

1. docker-logstash-forwarder must be run as root until Docker provides configurable ownership of shared volumes, because ```/var/lib/docker``` is owned by root on the host and mounted read only, so a non root user can not read from it ([docker#7918](https://github.com/docker/docker/issues/7198)).

2. The path of the containers content, on the hosts file system, has to be calculated by trying to take an educated guess based on your currently used docker driver since the docker folks consider this path internal and don't want to make it available via API ([docker#7915](https://github.com/docker/docker/issues/7915)). 

	Known to be working drivers are:
	
	* aufs
	* btrfs
	* devicemapper
	* overlay

Last but not least it probably should be mentioned, that this is the first time I wrote any go code (a few days, after work), so any 'Duh' pointers are greatly appreciated.

Pull Requests welcome :)
