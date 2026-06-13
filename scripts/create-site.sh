#!/bin/bash

i=$1
n=$2

rm -f /tmp/in_app$1 /tmp/out_app$1 /tmp/in_ctl$1 /tmp/out_ctl$1 /tmp/in_net$1 /tmp/out_net$1




echo "Creating Fifo"
mkfifo /tmp/in_net$i  /tmp/out_net$i

tee /tmp/in_net$i & PID1=$!
cat /tmp/out_net$i & PID2=$!

exec <> /tmp/in_net$i
exec <> /tmp/out_net$i

./bin/net -n "net$i" -id $i -ttin $PID1 -ttout $PID2 \
    < /tmp/in_net$i > /tmp/out_net$i

