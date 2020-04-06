#!/usr/bin/env bash

set -ue

syncer_log=$(kubectl logs vc-syncer-0 -n vc-manager)


echo "#podname,super_bind,create_vnode,just_bind_vpod,get_new_vpod" > $3.log
for i in $(seq $1 $2);
do
    # bind pod on super
    sb=$(cat "$syncer_log" | grep SUPER_BIND | grep "\[pod$i\]" | awk '{print $8}' | head -1)
    # create vnode on tenant
    cv=$(cat "$syncer_log" | grep CREATE_VNODE | grep "\[pod$i\]" | awk '{print $8}')
    # just bind vpod on tenant
    jbv=$(cat "$syncer_log" | grep JUST_BIND_VPOD | grep "\[pod$i\]" | awk '{print $8}')
    # get new vpod
    gnv=$(cat "$syncer_log" | grep GET_NEW_VPOD | grep "\[pod$i\]" | awk '{print $8}')
    echo "pod$i,$sb,$cv,$jbv,$gnv" >> $3.log
done
