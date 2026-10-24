#!/bin/bash

source functions.sh

source hadoop-env.sh

source cluster-config.sh

source yarn-env.sh


mkdir -p $HADOOP_HOME/hdfs/namenode


if [ ! -d $HADOOP_HOME/hdfs/namenode/current ]; then

    hdfs namenode -format
fi

mkdir -p $HADOOP_HOME/hdfs/datanode

echo "Iniciando NameNode..."
hdfs --daemon start namenode

echo "Iniciando ResourceManager..."
yarn --daemon start resourcemanager

echo "Iniciando NodeManager..."
yarn --daemon start nodemanager

echo "Iniciando datanode"
hdfs --daemon start datanode

tail -f /dev/null