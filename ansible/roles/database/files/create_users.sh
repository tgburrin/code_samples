#!/bin/bash

TOTAL=`psql -Xt -U postgres << EOF
select u.usename from pg_catalog.pg_user u where u.usename='pageview';
EOF`

TOTAL=`echo $TOTAL | sed 's/^\s\+|\s\+$//g'`
if [ "$TOTAL" == "" ]; then
  psql -U postgres << EOF
create user pageview with password 'password';
create database pageview with owner pageview;
\c pageview
drop schema public;
EOF

  psql -X -U pageview << EOF
create schema pageview;
EOF

  psql -X -U postgres pageview << EOF
CREATE EXTENSION "uuid-ossp" with SCHEMA pageview;
EOF
fi
