FROM qnib/uplain-golang

WORKDIR /usr/local/src/github.com/qnib/go-fisherman
COPY main.go .
COPY vendor/vendor.json vendor/vendor.json
RUN govendor fetch -v +m \
 && govendor install

FROM qnib/uplain-init

COPY --from=0 /usr/local/bin/go-fisherman /usr/local/bin/
CMD ["go-fisherman", "--help"]
