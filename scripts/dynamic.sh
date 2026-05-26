#!/bin/bash
 
# Generating named pipes, one input and one output per node
mkfifo in1 out1
mkfifo in2 out2
mkfifo in3 out3
mkfifo in4 out4
mkfifo in5 out5
mkfifo in6 out6
mkfifo in7 out7
 
# Starting the applications, one per node
./app --ident=node1 < in1 > out1 &
./app --ident=node2 < in2 > out2 &
./app --ident=node3 < in3 > out3 &
./app --ident=node4 < in4 > out4 &
./app --ident=node5 < in5 > out5 &
./app --ident=node6 < in6 > out6 &
./app --ident=node7 < in7 > out7 &
 
# Waiting for the link creation (security delay)
sleep 1 
 
# FIRST TOPOLOGY
# 1 -> 2 and 3
cat out1 | tee in2 in3 &
PID1=$!
 
# 2 -> 1, 4 and 5
cat out2 | tee in1 in4 in5 &
PID2=$!
 
# 3 -> 1 and 6
cat out3 | tee in1 in6 &
PID3=$!
 
# 4 -> 2, 5 and 7
cat out4 | tee in2 in5 in7 &
PID4=$!
 
# 5 -> 2 and 4
cat out5 | tee in2 in4 &
PID5=$!
 
# 6 -> 3 and 7
cat out6 | tee in3 in7 &
PID6=$!
 
# 7 -> 4 and 6
cat out7 | tee in4 in6 &
PID7=$!
 
# Waiting 10 seconds before changing the topology
sleep 10
 
# SECOND TOPOLOGY
# Deleting link 2->5
#  removing all links from 2
kill -KILL $PID2
#  adding links 2->1 and 2->4)
cat out2 | tee in1 in4 &
PID2=$!
 
# Deleting link 4->5
#  removing all links from 4
kill -KILL $PID4
#  adding links 4->2 and 4->7
cat out4 | tee in2 in7 &
PID4=$!
 
# Deleting link 5->2 and 5->4 (removing all links from 5)
kill -KILL $PID5
 
# Adding link 5->3
cat out5 | tee in3 &
PID5=$!
 
# Adding link from 3 towards 1, 5 and 6
#  removing all links from 3
kill -KILL $PID3
#  adding links 3->1, 3->5 and 3->6
cat out3 | tee in1 in5 in6 &
PID3=$!
 
 
# Waiting two seconds before changing the topology
sleep 2
 
# THIRD TOPOLOGY
# Deleting link 6 -> 7 and 7->6
#  removing all links from 6 and 7
kill -KILL $PID6 $PID7
#  adding link 6->3
cat out6 | tee in3 &
#  adding link 7->4
cat out7 | tee in4 &
 
# Waiting 20 seconds
sleep 20
 
# Killing all applications
killall app cat tee
 
# Deleting all named pipes
rm in* out*