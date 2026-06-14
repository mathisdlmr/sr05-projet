#!/bin/bash

i=$1
n=$2

rm -f /tmp/in_app$1 /tmp/out_app$1 /tmp/in_ctl$1 /tmp/out_ctl$1 /tmp/in_net$1 /tmp/out_net$1




echo "Creating Fifo"
mkfifo /tmp/in_app$i /tmp/out_app$i /tmp/in_ctl$i /tmp/out_ctl$i /tmp/in_net$i /tmp/out_net$i



#tee /tmp/in_net$i & PID1=$!
PID1=2334
cat /tmp/out_net$i & PID2=$!

echo "Cat pid is" $PID2

#exec <> /tmp/in_net$i
#exec <> /tmp/out_net$i

./bin/net -n "net$i" -id $i -ttout $PID2 \
    < /tmp/in_net$i > /tmp/out_net$i &

sleep 1
tee /tmp/in_net$i 