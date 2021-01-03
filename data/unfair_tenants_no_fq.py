#!/usr/bin/env python3

import matplotlib.pyplot as plt
import numpy as np

DATA_FN="unfair-tenants-no-fq.data.50"


counter = 0
with open(DATA_FN) as fp:
    lines = fp.readlines()
    x = []
    y = []
    for i, line in enumerate(lines):
        x.append(i) 
        y.append(float(line.split(",")[1]))
    
    plt.figure(figsize=(10,3))
    colors = {'Greedy user':'red', 'Normal user':'blue'}
    plt.ylim(0, 22)
    plt.yticks(np.arange(0, 24, 2), fontsize=16)
    # labels = list(colors.keys())
    # handles = [plt.Rectangle((0,0),1,1, color=colors[label]) for label in labels]
    bar_chart = plt.bar(x, y, edgecolor='black', linewidth=1, width=1)
    frame1 = plt.gca()
    frame1.axes.xaxis.set_visible(False)
    frame1.yaxis.grid(color='gray', linestyle='dashed')
    frame1.set_axisbelow(True)
    for i in range(10):
        bar_chart[i].set_color('r')
        bar_chart[i].set_edgecolor('black')
        bar_chart[i].set_width(1)
        bar_chart[i].set_linewidth(1)
    # plt.legend(handles, labels, fontsize=18, bbox_to_anchor=(0.5, 1.26), loc="upper center", ncol=2)
    plt.savefig("unfair-tenants-no-fq", bbox_inches='tight')
    # plt.show()
