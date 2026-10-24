#!/bin/bash

set -e

TEMP_FILE="/tmp/config-shanks-hosts"

# Cria backup apenas se ainda não existir
if [ ! -e "/etc/hosts.bak" ]; then
    cp /etc/hosts /etc/hosts.bak
fi

# Remove configuração antiga do SHANKS
sed '/# START S.H.A.N.K.S #/,/# END S.H.A.N.K.S #/d' \
    /etc/hosts > "$TEMP_FILE"

# Adiciona a nova configuração
{
    echo "# START S.H.A.N.K.S #"
    cat
    echo "# END S.H.A.N.K.S #"
} >> "$TEMP_FILE"

# Substitui o hosts
mv "$TEMP_FILE" /etc/hosts

chown root:root /etc/hosts
chmod 644 /etc/hosts