#!/bin/bash

mkfifo /tmp/F

./prog -n A < /tmp/F | ./prog -n B > /tmp/F