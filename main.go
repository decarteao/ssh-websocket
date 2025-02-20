package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

var current_connections int64 = 0 // aqui fica as conexoes

const (
	max_connections = 5000
	listen_ip       = "0.0.0.0"
	listen_port     = 80
	buffer_size     = 2048
	timeout_secs    = 60
)

func connect2ssh(ws net.Conn) {
	// Conectar ao servidor TCP
	conn, err := net.Dial("tcp", "localhost:22")
	if err != nil {
		fmt.Println("Erro ao se conectar no ssh")
		ws.Write([]byte("Erro ao se conectar no ssh"))
		return
	}
	defer func() {
		conn.Close()
	}()

	// Canal para enviar dados do SSH pro WebSocket
	go func() {
		// a cada iteracao
		buffer := make([]byte, buffer_size)
		for {
			conn.SetDeadline(time.Now().Add(timeout_secs * time.Second))
			n, err := conn.Read(buffer)
			if err != nil && err == io.EOF {
				fmt.Println("SSH Fechado")
				ws.Write([]byte(err.Error()))
				break
			}

			ws.Write(buffer[:n])
		}
	}()

	// Canal para enviar dados do WebSocket pro SSH
	buffer_ws := make([]byte, buffer_size)
	for {
		ws.SetDeadline(time.Now().Add(timeout_secs * time.Second))
		n, err := ws.Read(buffer_ws)

		if err != nil {
			if err != io.EOF {
				fmt.Printf("Erro depois da conexao: %s\n\n", err)
				continue
			} else {
				current_connections-- // diminuir na conexao
				fmt.Printf("Fechou conexão: %s\n\n", err)
				break
			}
		}

		// continue o fluxo
		conn.Write(buffer_ws[:n])
	}
}

func client_handler(w http.ResponseWriter, c *http.Request) {
	// verificar se o ip ta vir de angola
	if c.Header.Get("Cf-Ipcountry") != "AO" {
		fmt.Println("IP desconhecido!")
		return
	}

	// verificar se excedeu o maximo de conexao aceite
	if current_connections >= max_connections {
		http.Error(w, "Excedeu o maximo de conexao", 403)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Websocket nao suportado", 512)
		return
	}

	conn, rw, err := hj.Hijack()
	if err != nil {
		http.Error(w, "Erro ao ter controle total do fluxo TCP", 513)
		return
	}

	defer conn.Close() // fechar conexao

	// afirmar que a conexao esta sendo bem feita
	current_connections++

	// fazer o handshake
	fmt.Fprintf(rw, "HTTP/1.1 101 :) Ergam-se\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
	rw.Flush() // enviar tudo acima

	// criar o fluxo de dados com o ssh
	fmt.Println("[!] Nova conexao VPN")
	connect2ssh(conn)
}

func main() {
	fmt.Println("[!] Sung WebSocket iniciado")

	http.HandleFunc("/", client_handler)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", listen_ip, listen_port), nil)

	fmt.Printf("WS fechado: %s\n", err)
}
