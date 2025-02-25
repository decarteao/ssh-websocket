#!/bin/bash
apt update -y

# criar usuário sem acesso ao shell
USERNAME="sung"
PASSWORD="123.456"

sudo useradd -m -s /usr/sbin/nologin $USERNAME

# definir senha do usuário
echo "$USERNAME:$PASSWORD" | sudo chpasswd

# ignorar
#sudo useradd -m -s /usr/sbin/nologin sung
#echo "sung:123.456" | sudo chpasswd

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


# aumentar o maximo de sessions no ssh
sed -i 's/^#\?MaxSessions.*/MaxSessions 50000/' /etc/ssh/sshd_config
sed -i 's/^#\?MaxStartups.*/MaxStartups 50000:10:50000/' /etc/ssh/sshd_config

# aumentar o limite de conexoes abertas por processo
cp /etc/security/limits.conf /etc/security/limits.conf.bkp
bash -c 'echo -e "* soft nofile 1048576\n* hard nofile 1048576\n* soft nproc 1048576\n* hard nproc 1048576" >> /etc/security/limits.conf'

cp /etc/sysctl.conf /etc/sysctl.conf.bkp
bash -c 'echo -e "fs.file-max = 1048576\nnet.core.somaxconn = 65535\nnet.ipv4.tcp_max_syn_backlog = 65535\nnet.ipv4.ip_local_port_range = 1024 65535\nnet.ipv4.tcp_tw_reuse = 1\nnet.ipv4.tcp_fin_timeout = 15" >> /etc/sysctl.conf'

sysctl -p

# aumentar o limite de conexoes abertas no user e system
cp /etc/systemd/system.conf /etc/systemd/system.conf.bkp
sed -i '/DefaultLimitNOFILE/d' /etc/systemd/system.conf
sed -i '/DefaultLimitNPROC/d' /etc/systemd/system.conf
bash -c 'echo -e "DefaultLimitNOFILE=1048576\nDefaultLimitNPROC=1048576" >> /etc/systemd/system.conf'

cp /etc/systemd/user.conf /etc/systemd/user.conf.bkp
sed -i '/DefaultLimitNOFILE/d' /etc/systemd/user.conf
sed -i '/DefaultLimitNPROC/d' /etc/systemd/user.conf
bash -c 'echo -e "DefaultLimitNOFILE=1048576\nDefaultLimitNPROC=1048576" >> /etc/systemd/user.conf'

systemctl daemon-reexec

# ativar password no ssh
sed -i 's/^#\?PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config


# iniciar os services
systemctl daemon-reload

systemctl enable sshwebsocket
systemctl enable badvpn

systemctl restart sshwebsocket
systemctl restart badvpn
systemctl restart sshd

systemctl status sshwebsocket
systemctl status badvpn

# reinicializar
reboot
