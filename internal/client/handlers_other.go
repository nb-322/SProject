//go:build !windows

package client

import "errors"

func speak(text string) error {
	return errors.New("Not supported")
}
