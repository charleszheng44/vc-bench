#!/usr/bin/env python3

import matplotlib.pyplot as plt

total_latency_100_tenants = []

with open("100tenants10000pods/outData.diff") as fp:
    next(fp)
    lines = fp.readlines()
    for line in lines:
        tokens = line.split(',')         
        tl = tokens[len(tokens)-1]
        total_latency_100_tenants.append(int(tl))

total_latency_50_tenants = []

with open("50tenants10000pods/outData.diff") as fp:
    next(fp)
    lines = fp.readlines()
    for line in lines:
        tokens = line.split(',')         
        tl = tokens[len(tokens)-1]
        if int(tl) < 5: 
            total_latency_50_tenants.append(int(tl))
x_multis = [total_latency_100_tenants, total_latency_50_tenants]

plt.hist(x_multis, 10, edgecolor='black', linewidth=1.2)
labels = ["100 tenants", "50 tenants"]
plt.legend(labels)
plt.yscale('log')
plt.title('different sample sizes')

plt.show()
