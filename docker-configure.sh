#!/bin/bash


install_docker_based_arch() {

    source  /etc/os-release

    install_nvidia_driver_container "$ID"

    if command -v docker >/dev/null 2>&1; then
        echo 'Docker existe'
        return 0
    fi

    echo 'Docker não existe'

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


install_nvidia_driver_container() {


    if command -v nvidia-smi > /dev/null 2>&1; then

        case "$1" in
            ubuntu)
                downlaod_toolkit_ubuntu
                ;;

            rhel)
                download_toolkit_redhat
                ;;
            *)

        esac
    fi

    return
}

downlaod_toolkit_ubuntu() {

    apt-get update && sudo apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    gnupg2

    curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg \
    && curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
    sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
    tee /etc/apt/sources.list.d/nvidia-container-toolkit.list

    sed -i -e '/experimental/ s/^#//g' /etc/apt/sources.list.d/nvidia-container-toolkit.list

    apt-get update

    export NVIDIA_CONTAINER_TOOLKIT_VERSION=1.20.1-1
    apt-get install -y \
      nvidia-container-toolkit=${NVIDIA_CONTAINER_TOOLKIT_VERSION} \
      nvidia-container-toolkit-base=${NVIDIA_CONTAINER_TOOLKIT_VERSION} \
      libnvidia-container-tools=${NVIDIA_CONTAINER_TOOLKIT_VERSION} \
      libnvidia-container1=${NVIDIA_CONTAINER_TOOLKIT_VERSION}


    systemctl restart docker

}

download_toolkit_redhat() {

    dnf install -y \
    curl

    curl -s -L https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo | \
    tee /etc/yum.repos.d/nvidia-container-toolkit.repo

    dnf config-manager --enable nvidia-container-toolkit-experimental

    export NVIDIA_CONTAINER_TOOLKIT_VERSION=1.20.1-1
    dnf install -y \
      nvidia-container-toolkit-${NVIDIA_CONTAINER_TOOLKIT_VERSION} \
      nvidia-container-toolkit-base-${NVIDIA_CONTAINER_TOOLKIT_VERSION} \
      libnvidia-container-tools-${NVIDIA_CONTAINER_TOOLKIT_VERSION} \
      libnvidia-container1-${NVIDIA_CONTAINER_TOOLKIT_VERSION}

    systemctl restart docker
}
