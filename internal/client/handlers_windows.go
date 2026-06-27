package client

import (
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func speak(text string) error {
	ole.CoInitialize(0)
	defer ole.CoUninitialize()
	unknown, err := oleutil.CreateObject("SAPI.SpVoice")
	if err != nil {
		return err
	}
	voice, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer voice.Release()

	oleutil.CallMethod(voice, "Speak", text)
	return nil
}
