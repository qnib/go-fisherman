# go-fisherman
Golang library to help fishing information from moby


## Development


```bash
$ docker service create --name test --replicas=3 -p 8080:8080 qnib/httpcheck                                                                                                                                git:(master|●1✚2…
659c3qg8o5krw13kwrh7i4fvb
$ docker run -ti --name qframe --rm -e SKIP_ENTRYPOINTS=1 \
             -v ${GOPATH}/src/github.com/qnib/go-fisherman:/usr/local/src/github.com/qnib/go-fisherman \
             -w /usr/local/src/github.com/qnib/go-fisherman \
             qnib/uplain-golang bash
```
