#!/bin/bash

. /etc/profile.d/go.sh || exit 1

for pkg in github.com/lib/pq github.com/Shopify/sarama github.com/gorilla/handlers github.com/gorilla/mux github.com/jessevdk/go-flags github.com/satori/go.uuid
do
    echo "Getting ${pkg}"
    go get ${pkg} || exit 1
done

cd $GOPATH && mkdir -p src/github.com/tgburrin && cd src/github.com/tgburrin 
if [ -d rest_utilities ]; then
    cd rest_utilities && git fetch && git rebase
else
    git clone https://github.com/tgburrin/rest_utilities.git
fi
