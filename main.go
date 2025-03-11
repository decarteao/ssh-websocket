package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var current_connections int64 = 0 // aqui fica as conexoes
var max_connections int64 = 800

const (
	// configs gerais
	listen_ip         = "0.0.0.0"
	listen_port       = 80
	listen_ip_socks   = "127.0.0.1"
	listen_port_socks = 8999
	buffer_size       = 16384
	timeout_secs      = 30

	// autenticacao
	user     = "sung"
	password = "123.456"
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

	// afirmar que a conexao esta sendo bem feita
	atomic.AddInt64(&current_connections, 1)

	// diminuir na conexao ao fechar
	defer atomic.AddInt64(&current_connections, -1)

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
				log.Printf("Fechou conexão: %s\n\n", err)
				break
			}
		}

		// continue o fluxo
		conn.Write(buffer_ws[:n])
	}
}

func client_handler(conn net.Conn) {
	if atomic.LoadInt64(&current_connections) >= max_connections {
		conn.Write([]byte("HTTP/1.1 403 Servidor lotado\r\nServer: KaihoVPN\r\nMime-Version: 1.0\r\nContent-Type: text/html\r\n\r\n"))
		conn.Close()
		return
	}

	// ler payload
	payload := make([]byte, 512)
	conn.Read(payload)

	// log.Println(string(payload))
	if strings.Contains(string(payload), "GET /users HTTP/1.1") {
		// ver o total de users
		status := fmt.Sprintf(`{"data": "%d/%d"}`, atomic.LoadInt64(&current_connections), max_connections)
		conn.Write([]byte("HTTP/1.1 200 OK\r\nServer: KaihoVPN\r\nContent-Type: application/json\r\nContent-Length: " + fmt.Sprintf("%d", len([]byte(status))) + "\r\n\r\n" + status))
		conn.Close()
		return
	} else if !strings.Contains(strings.ToLower(string(payload)), "upgrade: websocket") {
		// validar payload para evitar spammers
		conn.Write([]byte("HTTP/1.1 403 Payload invalida\r\nServer: KaihoVPN\r\nMime-Version: 1.0\r\nContent-Type: text/html\r\n\r\nPayload Invalida :)"))
		conn.Close()
		return
	} else if !strings.Contains(strings.ToLower(string(payload)), fmt.Sprintf("\r\nuser: %s\r\n", user)) || !strings.Contains(strings.ToLower(string(payload)), fmt.Sprintf("\r\npassword: %s\r\n", password)) {
		// autenticar pela payload
		conn.Write([]byte("HTTP/1.1 403 Credenciais incorrectas\r\nServer: KaihoVPN\r\nMime-Version: 1.0\r\nContent-Type: text/html\r\n\r\nPayload Invalida :)"))
		conn.Close()
		return
	}

	// mandar o handshake
	conn.Write([]byte("HTTP/1.1 101 Ergam-se :)\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"))

	// iniciar o fluxo
	log.Println("\n{!} Nova conexão autenticada:", conn.RemoteAddr())

	// rodar o sub handler em uma goroutine
	go connect2ssh(conn)
}
func client_users(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"data": "%d/%d"}`, current_connections, max_connections)))
}

func main() {
	if len(os.Args) > 1 {
		val, err := strconv.ParseInt(os.Args[1], 10, 64)
		if err == nil {
			max_connections = val
		}
	}

	log.Printf("[!] MaxConnections: %d\n", max_connections)

	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", listen_ip, listen_port))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("[!] Sung WebSocket iniciado")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("erro no servidor:", err.Error())
		}

		go client_handler(conn)
	}
}
