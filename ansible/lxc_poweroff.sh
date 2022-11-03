#!/usr/bin/bash

ansible all -i inventory/lxc_hosts -m shell -a "/usr/sbin/poweroff"
