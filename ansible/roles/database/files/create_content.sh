#!/bin/bash

for c in testClientA testClientB testClientC testClientD testClientE
do
    CLIENT_ID=`psql -U pageview -XtAc "select * from create_client('${c}')"`
    for i in 01 02 03 04 05 06 07 08 09 10 11 12 13 14 15 16 17 18 19 20
    do
        psql -XtA -U pageview << EOF
select * from content_add('${CLIENT_ID}','${c}_${i}','https://${c,,}.com/${i}')
EOF
    done
done
