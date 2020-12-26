#!/usr/bin/env bash
for f in *;
do
    if [ -d $f ]; then
        cd $f
        for sub_f in *;
        do
            new_fn=$f.${sub_f##*.}
            mv $sub_f $new_fn
        done
        cd ../
    fi
done
