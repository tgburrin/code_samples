#!/bin/bash

. /etc/profile.d/postgres.sh

cd /installs || exit 1
if [ ! -d citus ]; then
  git clone https://github.com/citusdata/citus.git || exit 1
fi
cd citus && ./configure && make -j4 && make install
