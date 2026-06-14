#!/bin/bash

i=$1
n=$2

rm /tmp/in_net$i /tmp/out_net$i



mkfifo /tmp/in_net$i /tmp/out_net$i



echo "Starting cat"
cat /tmp/out_net$i & PID1=$!
sleep 1
echo "Starting net"
./bin/net -n net$i -id $i -next $n -ttout $PID1 \
        < /tmp/in_net$i > /tmp/out_net$i &
sleep 1
tee /tmp/in_net$i