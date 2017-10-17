#!/bin/bash

for c in testClientA testClientB testClientC testClientD testClientE
do
    CLIENT_ID=`psql -U tgburrin -t -A -c "select * from create_client('${c}')"`
    for i in 01 02 03 04 05 06 07 08 09 10 11 12 13 14 15 16 17 18 19 20
    do
        psql -t -A -U tgburrin << EOF
select * from content_add('${CLIENT_ID}','${c}_${i}','url://${c}/${i}')
EOF
    done
done
