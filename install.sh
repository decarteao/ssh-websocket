#!/bin/bash
apt update -y

# criar o user default da vpn
USERNAME="sung"
PASSWORD="123.456"

# Criar usuário sem acesso ao shell
sudo useradd -m -s /usr/sbin/nologin $USERNAME

# Definir senha do usuário
echo "$USERNAME:$PASSWORD" | sudo chpasswd

echo "Usuário '$USERNAME' criado sem acesso ao shell."

# instalar o sshwebsocket
wget -O sshwebsocket https://github.com/decarteao/ssh-websocket/raw/refs/heads/master/bin/sshwebsocket_ubuntu
chmod +x sshwebsocket

# instalar o service
cat > /etc/systemd/system/sshwebsocket.service <<-END
[Unit]
Description=SSHWebsocket
After=network.target

[Service]
ExecStart=/root/sshwebsocket
WorkingDirectory=/root/
StandardOutput=inherit
StandardError=inherit
Restart=always

[Install]
WantedBy=multi-user.target
END

systemctl enable sshwebsocket
systemctl restart sshwebsocket

# reinicializar
reboot
