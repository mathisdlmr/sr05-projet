#!/bin/bash

mkfifo /tmp/in_A
mkfifo /tmp/in_B
mkfifo /tmp/in_C

./prog -n A < /tmp/in_A | tee /tmp/in_B /tmp/in_C &
./prog -n B < /tmp/in_B | tee /tmp/in_A /tmp/in_C &
./prog -n C < /tmp/in_C | tee /tmp/in_A /tmp/in_B &

# Attente
echo "+ Attente de Crtl C pendant 1h..."
sleep 3600
./cleanup.sh