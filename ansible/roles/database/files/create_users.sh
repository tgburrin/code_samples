#!/bin/bash

TOTAL=`psql -t -U postgres << EOF
select u.usename from pg_catalog.pg_user u where u.usename='tgburrin';
EOF`

TOTAL=`echo $TOTAL | sed 's/^\s\+|\s\+$//g'`
if [ "$TOTAL" == "" ]; then
  psql -U postgres << EOF
create user tgburrin with password 'junk_password';
create database tgburrin with owner tgburrin;
\c tgburrin
drop schema public;
EOF

  psql -U tgburrin << EOF
create schema tgburrin;
EOF

  psql -U postgres tgburrin << EOF
CREATE EXTENSION "uuid-ossp" with SCHEMA tgburrin;
EOF
fi
