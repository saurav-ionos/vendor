#!/bin/sh

MAX_RESTART_COUNT=5

cd /logsvr 
i=0

while [ $i -lt $MAX_RESTART_COUNT ]
do
	echo Starting logsvr $i
	./logsvr $*
	i=`expr $i + 1`
	sleep 1
done

