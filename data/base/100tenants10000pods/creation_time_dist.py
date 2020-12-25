#!/usr/bin/env python3

import matplotlib.pyplot as plt

creation_ts = []
with open("100tenants10000pods.data") as fp:
    next(fp)
    lines = fp.readlines()
    for line in lines:
        tokens = line.split(",")
        creation_ts.append(int(tokens[1])) 

plt.hist(creation_ts, edgecolor='black', linewidth=1)
plt.show()
