package main

import (
	"bufio"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
)

const version = "20191008-5"

var (
	speedIn  float64
	speedOut float64
)

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
	SpeedIn     float64 `json:"speed_in"`
	SpeedOut    float64 `json:"speed_out"`
	Version     string  `json:"version"`
}

func messageGenerator() nodeMessage {
	var message nodeMessage
	message.SpeedIn = speedIn
	message.SpeedOut = speedOut
	message.Version = version
	if hostname, err := ioutil.ReadFile("/etc/hostname"); err == nil {
		message.Hostname = strings.TrimSuffix(string(hostname), "\n")
	}
	if stat, err := host.Info(); err == nil {
		if message.Hostname == "" {
			message.Hostname = stat.Hostname
		}
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

func networkSpeed() (bytesIn uint64, bytesOut uint64) {
	netStatsFile, err := os.Open("/proc/net/dev")
	if err != nil {
		panic(err)
	}
	defer netStatsFile.Close()
	reader := bufio.NewReader(netStatsFile)
	reader.ReadString('\n')
	reader.ReadString('\n')
	var line string
	for err == nil {
		line, err = reader.ReadString('\n')
		if line == "" {
			continue
		}
		splitLine := strings.Split(line, ":")
		interfaceName := splitLine[0]
		if strings.Contains(interfaceName, ".") {
			continue
		}
		if strings.HasPrefix(interfaceName, "eth") || strings.HasPrefix(interfaceName, "enp") {
			fields := strings.Fields(splitLine[1])
			bi, err := strconv.ParseUint(fields[0], 10, 64)
			if err != nil {
				bi = 0
			}
			bo, err := strconv.ParseUint(fields[8], 10, 64)
			if err != nil {
				bo = 0
			}
			bytesIn += bi
			bytesOut += bo
		}
	}
	return
}

func main() {
	go func() {
		bytesInOld, bytesOutOld := networkSpeed()
		for {
			time.Sleep(3 * time.Second)
			bytesIn, bytesOut := networkSpeed()
			speedIn = float64(bytesIn - bytesInOld) / 125000 / 3
			speedOut = float64(bytesOut - bytesOutOld) / 125000 / 3
			bytesInOld = bytesIn
			bytesOutOld = bytesOut
		}
	}()
	conn, _, err := websocket.DefaultDialer.Dial("wss://status.esd.cc/nws", nil)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
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
			case "ping":
				continue
			}
		}
	}
}
