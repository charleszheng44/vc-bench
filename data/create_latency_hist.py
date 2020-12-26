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
    plt.hist(x_multis, edgecolor='black', linewidth=1, bins=range(20), align='left')
    labels = ['baseline', '20 dws workers', '100 dws workers']
    plt.legend(labels)
    plt.ylim(1, 10000)
    plt.yscale('log')
    plt.xticks(np.arange(0, 19, 1))
    plt.title('{} pods creation latency of {} tenants'.format(num_pods, num_tenants))
    
    plt.show()

if __name__ == "__main__":
    plot_hist(100, 10000)
    plot_hist(100, 5000)
    plot_hist(100, 2500)
    plot_hist(100, 1250)
    plot_hist(50, 10000)
    plot_hist(50, 5000)
    plot_hist(50, 2500)
    plot_hist(50, 1250)
