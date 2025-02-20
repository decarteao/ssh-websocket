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

# instalar o badvpn-udp
wget -O badvpn_udp https://github.com/decarteao/ssh-websocket/raw/refs/heads/master/bin/badvpn_udp
chmod +x badvpn_udp

# instalar o service
cat > /etc/systemd/system/badvpn.service <<-END
[Unit]
Description=BadVPN
After=network.target

[Service]
ExecStart=/root/badvpn_udp
WorkingDirectory=/root/
StandardOutput=inherit
StandardError=inherit
Restart=always

[Install]
WantedBy=multi-user.target
END

# iniciar os services
systemctl daemon-reload

systemctl restart sshwebsocket
systemctl restart badvpn

# reinicializar
reboot
