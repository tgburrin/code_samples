#!/usr/bin/bash

BASE=`realpath $0`

export GOPATH=`dirname $BASE`/pageviewcount_service
echo "Using a base path of $GOPATH"

cd $GOPATH || exit 1
for p in github.com/Shopify/sarama github.com/gorilla/handlers github.com/gorilla/mux github.com/jessevdk/go-flags github.com/satori/go.uuid github.com/lib/pq
do
    go get ${p}
done

mkdir -p src/github.com/tgburrin/code_samples/golang  || exit 1
cd src/github.com/tgburrin/code_samples/golang || exit 1
for p in common dal_postgresql validation
do
    if [[ ! -L ${p} ]]; then
        ln -s ${GOPATH}/${p} .
    fi
done

cd $GOPATH
go fmt
go vet
go build
