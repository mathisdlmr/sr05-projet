#!/bin/bash

mkfifo /tmp/in_B
mkfifo /tmp/in_C

./prog -n B < /tmp/in_B &
./prog -n C < /tmp/in_C &

./prog -n A | tee /tmp/in_B /tmp/in_C &