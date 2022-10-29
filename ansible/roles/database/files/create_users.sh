#!/bin/bash

TOTAL=`psql -Xt -U postgres << EOF
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

  psql -X -U tgburrin << EOF
create schema tgburrin;
EOF

  psql -X -U postgres tgburrin << EOF
CREATE EXTENSION "uuid-ossp" with SCHEMA tgburrin;
EOF
fi
