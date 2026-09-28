#!/bin/bash


install_docker_based_arch() {
    if command -v docker >/dev/null 2>&1; then
        echo 'Docker existe'
        return 0
    fi

    echo 'Docker não existe'

    . /etc/os-release
    arch=$(dpkg --print-architecture)

    case "$ID" in
        ubuntu)
            docker_repo="ubuntu"
            ;;
        debian)
            docker_repo="debian"
            ;;
        raspbian)
            docker_repo="raspbian"
            ;;
        rhel)
            docker_repo="rhel"
            ;;
        *)
            echo "Distribuição não suportada: $ID"
            return 1
            ;;
    esac

    echo "Distribuição $ID"
    echo "Repositório Docker $docker_repo"

    apt-get update
    apt-get install -y ca-certificates curl
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL "https://download.docker.com/linux/${docker_repo}/gpg" -o /etc/apt/keyrings/docker.asc
    chmod a+r /etc/apt/keyrings/docker.asc

    cat >/etc/apt/sources.list.d/docker.sources <<EOF
Types: deb
URIs: https://download.docker.com/linux/
Suites: ${VERSION_CODENAME}
Components: stable
Architectures: ${arch}
Signed-By: /etc/apt/keyrings/docker.asc
EOF

    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

    groupadd docker
    usermod -aG docker "$USER"
    newgrp docker

}




