#!/usr/bin/env python3

import matplotlib.pyplot as plt
import numpy as np

DIR_20_WORKERS = "20dws200uws"
DIR_100_WORKERS = "100dws1000uws"

def draw_cpu_use(num_tenants):
    cpu_data_fn_20_workers = "{}/{}_tenants_cpu.dat".format(DIR_20_WORKERS, num_tenants)
    cpu_data_fn_100_workers = "{}/{}_tenants_cpu.dat".format(DIR_100_WORKERS, num_tenants)
    
    labels = ["1250", "2500", "5000", "10000"]
    cpu_times_20_workers = []
    with open(cpu_data_fn_20_workers) as fp:
        lines = fp.readlines()
        for line in lines:
            tokens = line.split(',')
            cpu_times_20_workers.append(float(tokens[1].rstrip()))

    cpu_times_100_workers = []
    with open(cpu_data_fn_100_workers) as fp:
        lines = fp.readlines()
        for line in lines:
            tokens = line.split(',')
            cpu_times_100_workers.append(float(tokens[1].rstrip()))
    
    y = np.arange(4)
    
    plt.figure(figsize=(6,3))
    plt.barh(y-0.125, cpu_times_20_workers, align='center', height=0.25)
    plt.barh(y+0.125, cpu_times_100_workers, align='center', height=0.25)
    plt.xlabel("CPU Time (second)", fontsize=16)
    plt.ylabel("Number of Pods", fontsize=16)
    plt.xticks(fontsize=14)
    plt.yticks(fontsize=14)
    plt.legend(["20 workers", "100 workers"], bbox_to_anchor=(0.5, 1.22), loc="upper center", fontsize=16, ncol=2)
    
    ax = plt.gca()
    ax.xaxis.grid(color='gray', linestyle='dashed')
    ax.set_axisbelow(True)

    plt.yticks(y, labels)
    
    # plt.show()
    fig_fn = "{}tenants_cpu.png".format(num_tenants)
    plt.savefig(fig_fn, bbox_inches='tight')

if __name__ == '__main__':
    draw_cpu_use(100)
    draw_cpu_use(50)
    draw_cpu_use(25)
