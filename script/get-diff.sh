#!/usr/bin/env bash

set -u
set -e 

cat $1 | awk -F ',' 'NR!=1{print $1","$3-$2","$4-$3}' > $1.diff
