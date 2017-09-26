#!/bin/bash

for i in 01 02 03 04 05 06 07 08 09 10 11 12 13 14 15 16 17 18 19 20
do
    psql -U tgburrin << EOF
insert into content select uuid_generate_v1(), 'test${i}','test${i}';
EOF
done
