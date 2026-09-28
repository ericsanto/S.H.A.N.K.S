#!/bin/bash


source docker-configure.sh
source zerotier-configure.sh


apt update && apt upgrade -y    

bin_directory="/usr/local/bin"

GROUP_NAME="shanks"

if getent group $GROUP_NAME > /dev/null 2>&1 ; then
    echo "Grupo shanks já existe"
else
    echo "Criando gupo $GROUP_NAME"

    groupadd $GROUP_NAME
    usermod -a -G $GROUP_NAME $SUDO_USER

    echo "Grupo $GROUP_NAME" criado com sucesso
fi

curl  https://raw.githubusercontent.com/ericsanto/S.H.A.N.K.S/staging/config-shanks-hosts.sh  -o config-hosts-shanks.sh && \
chmod +x config-hosts-shanks.sh && \
mv config-hosts-shanks.sh $bin_directory

chown -R root:root /usr/local/bin/config-hosts-shanks.sh

### DOWNLAOD ZEROTIER ###
install_zerotier

SUDOERS_TEMP=$(mktemp)

printf '%s\n' \
    '%shanks ALL=(root) NOPASSWD: /usr/local/bin/config-hosts-shanks.sh' \
    > "$SUDOERS_TEMP"

chmod 0440 "$SUDOERS_TEMP"

if visudo -cf "$SUDOERS_TEMP"; then
    install -o root -g root -m 0440 "$SUDOERS_TEMP" /etc/sudoers.d/shanks
else
    echo "Erro: regra sudoers inválida."
    rm -f "$SUDOERS_TEMP"
    exit 1
fi

rm -f "$SUDOERS_TEMP"

install_docker_based_arch
