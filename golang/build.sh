#!/usr/bin/bash

test -f /etc/profile.d/go.sh && . /etc/profile.d/go.sh

BASE=`realpath $0`
BASE=`dirname ${BASE}`

#export GOPATH=`dirname $BASE`/pageviewcount_service
echo "Using a base path of ${BASE} with go packages in $GOPATH"

cd $GOPATH || exit 1
for p in github.com/Shopify/sarama github.com/gorilla/handlers github.com/gorilla/mux github.com/jessevdk/go-flags github.com/satori/go.uuid github.com/lib/pq
do
    go get ${p}
done

cd ${BASE}/pageviewcount_service
go fmt
go vet
go build
