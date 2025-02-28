package main

import (
	"bufio"
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
var max_connections int64 = 50

const (
	listen_ip    = "0.0.0.0"
	listen_port  = 80
	buffer_size  = 2048
	timeout_secs = 60
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

	// rodar o handler em uma goroutine
	go func(conn net.Conn, rw *bufio.ReadWriter) {
		defer conn.Close()

		// fazer o handshake
		fmt.Fprint(rw, "HTTP/1.1 101 Ergam-se\r\n")
		fmt.Fprint(rw, "Upgrade: websocket\r\n")
		fmt.Fprint(rw, "Connection: Upgrade\r\n\r\n")
		rw.Flush()

		log.Println("[!] Nova conexao VPN")
		connect2ssh(conn)
	}(conn, rw)
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

	http.HandleFunc("/", client_handler)
	http.HandleFunc("/users", client_users)

	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", listen_ip, listen_port))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("[!] Sung WebSocket iniciado")
	log.Fatal(http.Serve(ln, nil))
}
