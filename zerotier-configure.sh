#!/bin/bash


install_zerotier() {

    if command -v zerotier-cli > /dev/null 2>&1; then
    
        echo "Zero Tier já está instalado no sistema"
        return 0

    fi

    
    curl -s 'https://raw.githubusercontent.com/zerotier/ZeroTierOne/main/doc/contact%40zerotier.com.gpg' | gpg --import && \
    if z=$(curl -s 'https://install.zerotier.com/' | gpg); then echo "$z" | sudo bash; fi

    systemctl start zerotier-one

}