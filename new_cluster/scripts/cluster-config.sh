#!/bin/bash

source functions.sh

MASTER_HOST=${MASTER_HOST:-master}

MEMORY=${MEMORY:-2048}

REPLICATION=${REPLICATION:-1}

HOSTNAME=$(hostname)

if [ "$HOSTNAME" = "master" ]; then

    YARN_NAME_ENV="master_YARN_LIMIT"
    YARN_CPU="master_YARN_LIMIT_CPU"

    capacity_scheduler=$(cat <<- EOM
<configuration>
    <property>
        <name>yarn.scheduler.capacity.resource-calculator</name>
        <value>org.apache.hadoop.yarn.util.resource.DominantResourceCalculator</value>
    </property>

    <property>
        <name>yarn.scheduler.capacity.root.queues</name>
        <value>default</value>
    </property>

    <property>
        <name>yarn.scheduler.capacity.root.default.capacity</name>
        <value>100</value>
    </property>
</configuration>
EOM
)

write_xml \
"$HADOOP_HOME/etc/hadoop/resource-types.xml" \
"$resource_type"
    write_xml \
    "$HADOOP_HOME/etc/hadoop/capacity-scheduler.xml" \
    "$capacity_scheduler"

else

    YARN_NAME_ENV="${HOSTNAME}_YARN_LIMIT"
    YARN_CPU="${HOSTNAME}_YARN_LIMIT_CPU"
fi

YARN_MEMORY="${!YARN_NAME_ENV:-2048}"
YARN_CPU="${!YARN_CPU:-2}"

GPU_YARN_ENV="${HOSTNAME}_GPU"
GPU_YARN="${!GPU_YARN_ENV:-false}"

if [ "$GPU_YARN" = "true" ]; then

    yarn=$(cat <<EOF
<configuration>

    <property>
        <name>yarn.resourcemanager.hostname</name>
        <value>${MASTER_HOST}</value>
    </property>

    <property>
        <name>yarn.nodemanager.aux-services</name>
        <value>mapreduce_shuffle</value>
    </property>

    <property>
        <name>yarn.nodemanager.resource.cpu-vcores</name>
        <value>${YARN_CPU}</value>
    </property>

    <property>
        <name>yarn.nodemanager.resource.memory-mb</name>
        <value>${YARN_MEMORY}</value>
    </property>

    <property>
        <name>yarn.log-aggregation-enable</name>
        <value>true</value>
    </property>

    <property>
        <name>yarn.nodemanager.resource-plugins</name>
        <value>yarn.io/gpu</value>
    </property>

    <property>
        <name>yarn.nodemanager.resource-plugins.gpu.allowed-gpu-devices</name>
        <value>auto</value>
    </property>

    <property>
        <name>yarn.nodemanager.resource-plugins.gpu.path-to-discovery-executables</name>
        <value>/usr/bin/nvidia-smi</value>
    </property>

    <property>
        <name>yarn.nodemanager.container-executor.class</name>
        <value>org.apache.hadoop.yarn.server.nodemanager.LinuxContainerExecutor</value>
    </property>

</configuration>
EOF
)

    write_xml \
    "$HADOOP_HOME/etc/hadoop/yarn-site.xml" \
    "$yarn"

else

    yarn=$(cat <<EOF
<configuration>

    <property>
        <name>yarn.resourcemanager.hostname</name>
        <value>${MASTER_HOST}</value>
    </property>

    <property>
        <name>yarn.nodemanager.aux-services</name>
        <value>mapreduce_shuffle</value>
    </property>

    <property>
        <name>yarn.nodemanager.resource.cpu-vcores</name>
        <value>${YARN_CPU}</value>
    </property>

    <property>
        <name>yarn.nodemanager.resource.memory-mb</name>
        <value>${YARN_MEMORY}</value>
    </property>

    <property>
        <name>yarn.log-aggregation-enable</name>
        <value>true</value>
    </property>

</configuration>
EOF
)

    write_xml \
    "$HADOOP_HOME/etc/hadoop/yarn-site.xml" \
    "$yarn"

fi

core=$(cat <<EOF
<configuration>

    <property>
        <name>fs.defaultFS</name>
        <value>hdfs://${MASTER_HOST}:9000</value>
    </property>

</configuration>
EOF
)

write_xml \
"$HADOOP_HOME/etc/hadoop/core-site.xml" \
"$core"

hdfs=$(cat <<EOF
<configuration>

    <property>
        <name>dfs.namenode.name.dir</name>
        <value>$HADOOP_HOME/hdfs/namenode</value>
    </property>

    <property>
        <name>dfs.datanode.data.dir</name>
        <value>$HADOOP_HOME/hdfs/datanode</value>
    </property>

    <property>
        <name>dfs.replication</name>
        <value>${REPLICATION}</value>
    </property>

</configuration>
EOF
)

write_xml \
"$HADOOP_HOME/etc/hadoop/hdfs-site.xml" \
"$hdfs"

cat workers.txt >> "$HADOOP_HOME/etc/hadoop/workers"

mapred=$(cat <<- EOM
<configuration>

    <property>
        <name>mapreduce.framework.name</name>
        <value>yarn</value>
    </property>

    <property>
        <name>yarn.app.mapreduce.am.env</name>
        <value>HADOOP_MAPRED_HOME=/home/hadoop/hadoop</value>
    </property>

    <property>
        <name>mapreduce.map.env</name>
        <value>HADOOP_MAPRED_HOME=/home/hadoop/hadoop</value>
    </property>

    <property>
        <name>mapreduce.reduce.env</name>
        <value>HADOOP_MAPRED_HOME=/home/hadoop/hadoop</value>
    </property>

</configuration>
EOM
)

write_xml \
"$HADOOP_HOME/etc/hadoop/mapred-site.xml" \
"$mapred"

resource_type=$(cat <<- EOM
<configuration>
    <property>
        <name>yarn.resource-types</name>
        <value>yarn.io/gpu</value>
    </property>
</configuration>
EOM
)

write_xml \
"$HADOOP_HOME/etc/hadoop/resource-types.xml" \
"$resource_type"