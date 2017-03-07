#!/bin/sh

MAX_RESTART_COUNT=5

cd /apisvr 
i=0

while [ $i -lt $MAX_RESTART_COUNT ]
do
	echo Starting apisvr $i
	./apiui  $*
	i=`expr $i + 1`
	sleep 1
done

