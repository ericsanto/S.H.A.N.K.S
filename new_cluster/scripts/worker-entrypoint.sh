#!/bin/bash

source functions.sh

source hadoop-env.sh

source cluster-config.sh

source yarn-env.sh

mkdir -p $HADOOP_HOME/hdfs/datanode
chown hadoop:hadoop $HADOOP_HOME/hdfs/datanode

echo "Iniciando DataNode..."
hdfs --daemon start datanode

echo "Iniciando NodeManager..."
yarn --daemon start nodemanager

tail -f /dev/null