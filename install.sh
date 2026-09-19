#!/bin/bash

bin_directory="/usr/local/bin"

GROUP_NAME="shanks"

groupadd $GROUP_NAME

usermod -a -G $GROUP_NAME $SUDO_USER

curl  https://raw.githubusercontent.com/ericsanto/S.H.A.N.K.S/feat/installApp/config-shanks-hosts.sh  -o config-hosts-shanks.sh && \
chmod +x config-hosts-shanks.sh && \
mv config-hosts-shanks.sh $bin_directory

chown -R root:root /usr/local/bin/config-hosts-shanks.sh


### DOWNLAOD ZEROTIER ###
curl -s 'https://raw.githubusercontent.com/zerotier/ZeroTierOne/main/doc/contact%40zerotier.com.gpg' | gpg --import && \
if z=$(curl -s 'https://install.zerotier.com/' | gpg); then echo "$z" | sudo bash; fi


systemctl start zerotier-one


### LEMBRAR DE ADICIONAR O SHANKS NO SUDOERS ###

SUDOERS_TEMP=$(mktemp)

printf '%s\n' \
    '%shanks ALL=(root) NOPASSWD: /usr/local/bin/config-hosts-shanks.sh' \
    > "$SUDOERS_TEMP"

chmod 0440 "$SUDOERS_TEMP"

if visudo -cf "$SUDOERS_TEMP"; then
    install -o root -g root -m 0440 \   
        "$SUDOERS_TEMP" \
        /etc/sudoers.d/shanks
else
    echo "Erro: regra sudoers inválida."
    rm -f "$SUDOERS_TEMP"
    exit 1
fi

rm -f "$SUDOERS_TEMP"


