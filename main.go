package main

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

func main() {
	file, _ := os.Open("conf.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	configuration = Config{}
	err := decoder.Decode(&configuration)
	if err != nil {
		log.Fatal(err)
	}

	DivertInit()
	IPWhiteList = make(map[string]int64)

	Handle, err := WinDivertOpen("true", 0, 1000, 0)
	if err != nil {
		log.Fatal(err)
	}
	go PXLoop(Handle)
	go TXLoop(Handle)
	go RXLoop(Handle)
	for !EndFlag {
		time.Sleep(1000)
	}
	WinDivertShutdown(Handle, 0x3)
	WinDivertClose(Handle)
}
