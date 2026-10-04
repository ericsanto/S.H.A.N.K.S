#!/usr/bin/bash

set -euo pipefail
: "${SPARK_LOCAL_IP:?Defina SPARK_LOCAL_IP no environment do container}"
mkdir -p "$SPARK_CONF_DIR"
cat > "$SPARK_CONF_DIR/spark-defaults.conf" <<EOF
spark.master yarn
spark.submit.deployMode client
spark.driver.host ${SPARK_LOCAL_IP}
spark.driver.bindAddress 0.0.0.0
spark.driver.port 37001
spark.blockManager.port 37002
EOF
# Token de autenticação padrão do Jupyter permanece habilitado.
exec start-notebook.py \
  --ServerApp.ip="$SPARK_LOCAL_IP" \
  --ServerApp.port=8888 \
  --ServerApp.port_retries=0 \
  --ServerApp.open_browser=False \
  --ServerApp.root_dir=/home/jovyan/work