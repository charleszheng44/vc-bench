#!/usr/bin/env python3

import matplotlib.pyplot as plt
import numpy as np

DIR_20_WORKERS = "20dws200uws"
DIR_100_WORKERS = "100dws1000uws"
DIR_BASE = "base"

# plot_hist
def plot_hist(num_tenants, num_pods):
    """ plot_hist plots the histogram of the configuration with num_tenants 
    and num_pods. The hitogram will contain three datasets: 20 dws workers,
    100 dws workers and native(creating pod on the super master directly)
    """
    
    fn_prefix_20_workers = "{0}/{1}tenants{2}pods/{1}tenants{2}pods".format(
            DIR_20_WORKERS, num_tenants, num_pods)
    fn_prefix_100_workers = "{0}/{1}tenants{2}pods/{1}tenants{2}pods".format(
            DIR_100_WORKERS, num_tenants, num_pods)
    fn_prefix_base = "{0}/{1}tenants{2}pods/{1}tenants{2}pods".format(
            DIR_BASE, num_tenants, num_pods)
    
    # generates the dataset for the 20 workers
    total_latency_20_workers = []
    with open("{}.diff".format(fn_prefix_20_workers)) as fp:
        next(fp)
        lines = fp.readlines()
        for line in lines:
            tokens = line.split(',')         
            tl = tokens[len(tokens)-1]
            total_latency_20_workers.append(int(tl))

    # generates the dataset for the 100 workers
    total_latency_100_workers = []
    with open("{}.diff".format(fn_prefix_100_workers)) as fp:
        next(fp)
        lines = fp.readlines()
        for line in lines:
            tokens = line.split(',')         
            tl = tokens[len(tokens)-1]
            total_latency_100_workers.append(int(tl))

    # generates the dataset for the baseline
    total_latency_base = []
    with open("{}.diff".format(fn_prefix_base)) as fp:
        next(fp)
        lines = fp.readlines()
        for line in lines:
            tokens = line.split(',')         
            tl = tokens[len(tokens)-1]
            total_latency_base.append(int(tl))

    x_multis = [total_latency_base, 
            total_latency_20_workers, 
            total_latency_100_workers]

    plt.figure(figsize=(10, 5))
    plt.hist(x_multis, edgecolor='black', linewidth=1, bins=np.arange(0, 16, 2), align='left')
    labels = ['Baseline', '20 workers', '100 workers']
    if num_pods == 1250:
        plt.legend(labels,bbox_to_anchor=(0.5, 1.22), loc="upper center", fontsize=18, ncol=len(labels))
    
    ylim = 3000
    if num_pods == 1250:
        ylim = 1500
        plt.yticks(np.arange(0, 1600, 300), fontsize=14)
    
    if num_pods == 2500:
        ylim = 3000

    if num_pods == 5000:
        ylim = 5000

    if num_pods == 10000:
        ylim = 5000

    # ylim = 10000
    plt.ylim(1, ylim)
    # plt.yscale('log')
    plt.xlabel('Time Bucket (seconds)', fontsize=18)
    plt.ylabel('Number of Pods', fontsize=18)
    xlabels = ["0", "2", "4", "6", "8", "10", "<18", ""]
    plt.xticks(np.arange(0, 16, 2), xlabels, fontsize=16)
    plt.yticks(fontsize=16)
    # plt.title('{} pods creation latency of {} tenants'.format(num_pods, num_tenants), fontdict = {'fontsize' : 22})
    
    fig_fn = "{}tenants{}pods.png".format(num_tenants, num_pods)
    ax = plt.gca()
    ax.yaxis.grid(color='gray', linestyle='dashed')
    ax.set_axisbelow(True)
    plt.text(4, ylim*0.85, "\n{} Pods".format(num_pods), fontsize=22, weight='bold')
    xticks = ax.xaxis.get_major_ticks()
    xticks[-1].label1.set_visible(False)
    # plt.show()
    plt.savefig(fig_fn, bbox_inches='tight')

if __name__ == "__main__":
    plot_hist(100, 10000)
    plot_hist(100, 5000)
    plot_hist(100, 2500)
    plot_hist(100, 1250)
    plot_hist(50, 10000)
    plot_hist(50, 5000)
    plot_hist(50, 2500)
    plot_hist(50, 1250)
    plot_hist(25, 10000)
    plot_hist(25, 5000)
    plot_hist(25, 2500)
    plot_hist(25, 1250)
