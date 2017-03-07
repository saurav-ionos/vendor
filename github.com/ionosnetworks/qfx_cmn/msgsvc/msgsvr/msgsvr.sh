#!/bin/sh

MAX_RESTART_COUNT=5

if [ -z $ETCD_CLUSTER_IP ]
then
    echo "etcd" > ETCD_CLUSTER_IP
else
    echo $ETCD_CLUSTER_IP > ETCD_CLUSTER_IP
fi

/msgsvr/demuxer.sh &

cd /msgsvr 
i=0

while [ $i -lt $MAX_RESTART_COUNT ]
do
	echo Starting msgsvr $i
	./msgsvr $*
	i=`expr $i + 1`
	sleep 1
done

