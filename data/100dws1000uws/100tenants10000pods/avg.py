#!/usr/bin/env python3

import statistics as st

with open("100tenants10000pods.diff") as fp:
    next(fp)
    lines = fp.readlines()
    
    dwsQDelay, dwsProcessDelay, superCreationDeplay, uwsQDelay, tenantUpdateDelay, total = [], [], [], [], [], []
    for line in lines:
        line = line.split(',')
        dwsQDelay.append(int(line[1]))
        dwsProcessDelay.append(int(line[2]))
        superCreationDeplay.append(int(line[3]))
        uwsQDelay.append(int(line[4]))
        tenantUpdateDelay.append(int(line[5]))
        total.append(int(line[6]))

    print('dwsQDelay {0}'.format(st.mean(dwsQDelay))) 
    print('dwsProcessDelay {0}'.format(st.mean(dwsProcessDelay))) 
    print('superCreationDeplay {0}'.format(st.mean(superCreationDeplay))) 
    print('uwsQDelay {0}'.format(st.mean(uwsQDelay))) 
    print('tenantUpdateDelay {0}'.format(st.mean(tenantUpdateDelay))) 
    print('total {0}'.format(st.mean(total))) 

