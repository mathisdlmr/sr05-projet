#!/bin/bash

mkfifo /tmp/in_A /tmp/in_B /tmp/in_C /tmp/in_D
mkfifo /tmp/out_A /tmp/out_B /tmp/out_C /tmp/out_D

./prog -n A < /tmp/in_A > /tmp/out_A &
./prog -n B < /tmp/in_B > /tmp/out_B &
./prog -n C < /tmp/in_C > /tmp/out_C &
./prog -n D < /tmp/in_D > /tmp/out_D &

cat /tmp/out_A > /tmp/in_B &
cat /tmp/out_B | tee /tmp/in_C > /tmp/in_D &
cat /tmp/out_C > /tmp/in_D &
cat /tmp/out_D > /tmp/in_A &

# Attente
echo "+ Attente de Crtl C pendant 1h..."
sleep 3600
./cleanup.sh