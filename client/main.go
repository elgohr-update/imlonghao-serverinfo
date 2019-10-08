package main

import (
	"io/ioutil"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
)

const version = "20191008-1"

type nodeMessage struct {
	Hostname    string  `json:"hostname"`
	Platform    string  `json:"platform"`
	Kernel      string  `json:"kernel"`
	Uptime      uint64  `json:"uptime"`
	Load1       float64 `json:"load1"`
	Load5       float64 `json:"load5"`
	Load15      float64 `json:"load15"`
	MemoryTotal uint64  `json:"memory_total"`
	MemoryUsed  uint64  `json:"memory_used"`
	SwapTotal   uint64  `json:"swap_total"`
	SwapUsed    uint64  `json:"swap_used"`
	DiskTotal   uint64  `json:"disk_total"`
	DiskUsed    uint64  `json:"disk_used"`
	Version     string  `json:"version"`
}

func messageGenerator() nodeMessage {
	var message nodeMessage
	message.Version = version
	if hostname, err := ioutil.ReadFile("/etc/hostname"); err == nil {
		message.Hostname = strings.TrimSuffix(string(hostname), "\n")
	}
	if stat, err := host.Info(); err == nil {
		message.Platform = stat.Platform
		message.Kernel = stat.KernelVersion
		message.Uptime = stat.Uptime
	}
	if stat, err := load.Avg(); err == nil {
		message.Load1 = stat.Load1
		message.Load5 = stat.Load5
		message.Load15 = stat.Load15
	}
	if stat, err := mem.VirtualMemory(); err == nil {
		message.MemoryTotal = stat.Total
		message.MemoryUsed = stat.Used
	}
	if stat, err := mem.SwapMemory(); err == nil {
		message.SwapTotal = stat.Total
		message.SwapUsed = stat.Used
	}
	if stat, err := disk.Usage("/"); err == nil {
		message.DiskTotal = stat.Total
		message.DiskUsed = stat.Used
	}
	return message
}

func main() {
	conn, _, err := websocket.DefaultDialer.Dial("wss://status.esd.cc/nws", nil)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	conn.SetPingHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(10 * time.Second)); return nil })
	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			panic(err)
		}
		switch mt {
		case websocket.TextMessage:
			switch string(message) {
			case "check":
				if err := conn.WriteJSON(messageGenerator()); err != nil {
					panic(err)
				}
			}
		}
	}
}
