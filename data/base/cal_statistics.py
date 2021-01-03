#!/usr/bin/env python3

import statistics
import numpy as np

def cal_statistics(num_tenants, num_pods):
    fn = "{0}tenants{1}pods/{0}tenants{1}pods.data".format(num_tenants, num_pods)
    start_times = []
    end_times = []
    time_elapses = []
    with open (fn) as fp:
        next(fp)
        for line in fp.readlines():
            tokens = line.split(",")
            start_times.append(int(tokens[1]))
            end_times.append(int(tokens[2]))
            time_elapses.append(int(tokens[2]) - int(tokens[1]))
        st = min(start_times)
        et = max(end_times)
        total_time = et - st
        throughput = num_pods / total_time
        np_arr = np.array(time_elapses) 
        print ("""
 ====================================================
 {}Tenants{}Pods
 Total Time = {}
 Throughput = {}
 Max Creation Time = {}
 Min Creation Time = {}
 Average Creation Time = {}
 99% Creation Time = {}
 """.format(
        num_tenants,
        num_pods,
        total_time, 
        throughput, 
        max(time_elapses), 
        min(time_elapses), 
        statistics.mean(time_elapses),
        np.percentile(np_arr, 99)))

if __name__ == '__main__':
    cal_statistics(25, 1250)
    cal_statistics(50, 1250)
    cal_statistics(100, 1250)
    cal_statistics(25, 2500)
    cal_statistics(50, 2500)
    cal_statistics(100, 2500)
    cal_statistics(25, 5000)
    cal_statistics(50, 5000)
    cal_statistics(100, 5000)
    cal_statistics(25, 10000)
    cal_statistics(50, 10000)
    cal_statistics(100, 10000)
