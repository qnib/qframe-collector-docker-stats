# qframe-collector-docker-stats
Qframe collector to spawn metric goroutine for each running container.

This plugin will send each stat (once a second) as `QMsg.Data` down the Data channel. Thus, it needs to be processed, most likely in a `qframe-filter-docker-stats`.


## Development
```bash
$ docker run -ti --name qframe-collector-docker-events --rm -e SKIP_ENTRYPOINTS=1 \
             -v ${GOPATH}/src/github.com/qnib/qframe-collector-docker-stats:/usr/local/src/github.com/qnib/qframe-collector-docker-stats \
             -v ${GOPATH}/src/github.com/qnib/qframe-collector-docker-events:/usr/local/src/github.com/qnib/qframe-collector-docker-events \
             -v ${GOPATH}/src/github.com/qnib/qframe-filter-id:/usr/local/src/github.com/qnib/qframe-filter-id \
             -v ${GOPATH}/src/github.com/qnib/qframe-types:/usr/local/src/github.com/qnib/qframe-types \
             -v ${GOPATH}/src/github.com/qnib/qframe-utils:/usr/local/src/github.com/qnib/qframe-utils \
             -v /var/run/docker.sock:/var/run/docker.sock \
             -w /usr/local/src/github.com/qnib/qframe-collector-docker-stats \
              qnib/uplain-golang bash
root@78e72f2b5bcd:# govendor update github.com/qnib/qframe-types github.com/qnib/qframe-utils github.com/qnib/qframe-filter-id/lib github.com/qnib/qframe-collector-docker-events/lib
```
