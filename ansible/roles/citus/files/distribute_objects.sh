#!/bin/bash

psql -U tgburrin << EOF
drop index by_url;
SELECT create_distributed_table('content','id');
SELECT create_distributed_table('content_counter','id');
EOF;
