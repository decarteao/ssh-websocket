package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var current_connections int64 = 0 // aqui fica as conexoes

const (
	max_connections = 15000
	listen_ip       = "0.0.0.0"
	listen_port     = 80
	buffer_size     = 2048
	timeout_secs    = 60
)

func connect2ssh(ws net.Conn) {
	// Conectar ao servidor TCP
	conn, err := net.DialTimeout("tcp", "localhost:22", timeout_secs*time.Second)
	if err != nil {
		log.Println("Erro ao se conectar no ssh")
		ws.Write([]byte("Erro ao se conectar no ssh"))
		return
	}

	// fechar conexao dps que tudo se encerrar
	defer conn.Close()

	// Canal para enviar dados do SSH pro WebSocket
	go func() {
		// a cada iteracao
		buffer := make([]byte, buffer_size)
		for {
			conn.SetDeadline(time.Now().Add(timeout_secs * time.Second))
			n, err := conn.Read(buffer)
			if err != nil && err == io.EOF {
				log.Println("SSH Fechado")
				ws.Write([]byte(err.Error()))
				break
			}
			if err == nil {
				ws.Write(buffer[:n])
			}
		}
	}()

	// Canal para enviar dados do WebSocket pro SSH
	buffer_ws := make([]byte, buffer_size)
	for {
		ws.SetDeadline(time.Now().Add(timeout_secs * time.Second))
		n, err := ws.Read(buffer_ws)

		if err != nil {
			if err != io.EOF {
				log.Printf("Erro depois da conexao: %s\n\n", err)
				continue
			} else {
				atomic.AddInt64(&current_connections, -1) // diminuir na conexao
				log.Printf("Fechou conexão: %s\n\n", err)
				break
			}
		}

		// continue o fluxo
		conn.Write(buffer_ws[:n])
	}
}

func client_handler(w http.ResponseWriter, c *http.Request) {
	// verificar se excedeu o maximo de conexao aceite
	if atomic.LoadInt64(&current_connections) >= max_connections {
		http.Error(w, "Excedeu o maximo de conexao", 403)
		return
	}

	// validar payload antes de iniciar sessao
	if strings.ToLower(c.Header.Get("Upgrade")) != "websocket" {
		http.Error(w, "Payload invalida", 503)
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
	atomic.AddInt64(&current_connections, 1)

	// fazer o handshake
	fmt.Fprint(rw, "HTTP/1.1 101 Ergam-se\r\n")
	fmt.Fprint(rw, "Upgrade: websocket\r\n")
	fmt.Fprint(rw, "Connection: Upgrade\r\n\r\n")
	rw.Flush() // enviar tudo acima

	// criar o fluxo de dados com o ssh
	log.Println("[!] Nova conexao VPN")
	connect2ssh(conn)
}

func main() {
	log.Println("[!] Sung WebSocket iniciado")

	http.HandleFunc("/", client_handler)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", listen_ip, listen_port), nil)

	log.Printf("WS fechado: %s\n", err)
}
