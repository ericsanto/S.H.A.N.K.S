#!/bin/bash


install_docker_based_arch() {

    source  /etc/os-release

    if command -v docker >/dev/null 2>&1; then
        echo 'Docker existe'
    else
        echo 'Docker não existe'

        case "$ID" in
            garuda|arch|manjaro)
                pacman -Syu --noconfirm --needed docker docker-buildx docker-compose
                ;;
            ubuntu|debian|raspbian)
                arch=$(dpkg --print-architecture)
                codename="${UBUNTU_CODENAME:-$VERSION_CODENAME}"
                docker_repo="$ID"

                echo "Distribuição $ID"
                echo "Repositório Docker $docker_repo"

                apt-get update
                apt-get install -y ca-certificates curl
                install -m 0755 -d /etc/apt/keyrings
                curl -fsSL "https://download.docker.com/linux/${docker_repo}/gpg" -o /etc/apt/keyrings/docker.asc
                chmod a+r /etc/apt/keyrings/docker.asc

                cat >/etc/apt/sources.list.d/docker.sources <<EOF
Types: deb
URIs: https://download.docker.com/linux/${docker_repo}
Suites: ${codename}
Components: stable
Architectures: ${arch}
Signed-By: /etc/apt/keyrings/docker.asc
EOF

                apt-get update
                apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
                ;;
            *)
                echo "Distribuição não suportada: $ID"
                return 1
                ;;
        esac
    fi

    groupadd -f docker

    docker_user="${SUDO_USER:-$USER}"
    if id "$docker_user" >/dev/null 2>&1; then
        usermod -aG docker "$docker_user"
    else
        echo "Usuário não encontrado para adicionar ao grupo docker: $docker_user"
        return 1
    fi

    systemctl enable --now docker
    install_nvidia_driver_container "$ID"
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
            arch|garuda|manjaro)
                download_toolkit_arch
                ;;
            *)
                echo "NVIDIA Container Toolkit não suportado para a distribuição: $1"
                return 1

        esac

        if ! command -v nvidia-ctk >/dev/null 2>&1; then
            echo "nvidia-ctk não foi encontrado após a instalação"
            return 1
        fi

        nvidia-ctk runtime configure --runtime=docker
        systemctl restart docker
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

}

download_toolkit_arch() {
    pacman -S --noconfirm --needed nvidia-container-toolkit
}
