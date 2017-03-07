#!/bin/sh

MAX_RESTART_COUNT=5

sleep 5
if [ -z $SECONDAY_CONTROLLER ]
then

  i=0

  while [ $i -lt $MAX_RESTART_COUNT ]
  do
      echo Starting demux 
       /msgsvr/demuxer 
       i=`expr $i + 1`
       sleep 1
  done

fi

