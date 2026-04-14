#!/bin/bash

mkfifo /tmp/in_A /tmp/in_B /tmp/in_C
mkfifo /tmp/out_A /tmp/out_B /tmp/out_C

./prog -n A < /tmp/in_A > /tmp/out_A &
./prog -n B < /tmp/in_B > /tmp/out_B &
./prog -n C < /tmp/in_C > /tmp/out_C &

cat /tmp/out_A > /tmp/in_B &
cat /tmp/out_B > /tmp/in_C &
cat /tmp/out_C > /tmp/in_A &

# Attente
echo "+ Attente de Crtl C pendant 1h..."
sleep 3600
./cleanup.sh