#!/bin/bash

if [[ "$1" == "$2" ]]; then
  echo "Not adding master to itself"
  exit 0
fi

psql -U postgres tgburrin -c "SELECT * from master_add_node('$2', 5432);"
