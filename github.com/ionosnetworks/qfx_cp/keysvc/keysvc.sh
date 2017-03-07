#!/bin/sh

MAX_RESTART_COUNT=5

cd /keysvc 
i=0

while [ $i -lt $MAX_RESTART_COUNT ]
do
	echo Starting keysvc $i
	./keysvc $*
	i=`expr $i + 1`
	sleep 1
done

