#!/usr/bin/env bash

declare -ir NUM_PO=${NUM_POD:-10}
declare -A PO_CRT_TIME
declare -r PO_CRT_STAT_FN=${PO_CRT_STAT_FN:-po-create-time.data}
declare -ir NUM_VC=${NUM_VC:-10}
PO_TEMP="
"

# get kubeconfig of each vc

# equally spread pods to vc
PO_PER_VC=$((NUM_PO/NUM_VC))

# record the start time of creation
PO_YAML=$(echo $PO_TEMP | sed "s/_po_name/$PO_NAME$i/")
PO_CRT_TIME[$PO_NAME$i]=$(date +%s) && echo $PO_YAML | kubectl apply -f - 

# periodically(10 seconds) check if there are newly created pods, get the 
# complete time of creation by checking the `creationTimestamp`, and 
# caculate the creation latency 

while : ; do
    [ ${#PO_CRT_TIME[@]} -eq ] && break
    kubectl get po $po_name -o jsonpath='{.metadata.creationTimestamp}' --kubeconfig $kb_cfg
    sleep 10
done

# write the record to file









