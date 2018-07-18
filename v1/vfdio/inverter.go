// Copyright (c) 2018 Sebastian Schleemilch
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package vfdio

import (
	"encoding/binary"
	"fmt"
	"github.com/jacobsa/go-serial/serial"
	"github.com/npat-efault/crc16"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HyInverter struct {
	port            io.ReadWriteCloser
	hash16          crc16.Hash16
	stop            bool
	once            sync.Once
	cmdChannel      chan string
	outputFrequency uint16
	outputRpm       uint16
	lastReceived    time.Time
}

const RequestInverterIntervalMillis int64 = 750

// Experimentally determined with Inverter display.
// May be different on other setups.
const inverterHertzPerRpm float32 = 3.47222
const maxRpm = 11520 // Experimentally determined with inverter

// Open creates a serial port handle for the Hy. Inverter.
// Example portName: /dev/ttyUSB0
func (o *HyInverter) Open(portName string) (err error) {
	o.once.Do(func() {
		options := serial.OpenOptions{
			PortName:        portName,
			BaudRate:        9200,
			DataBits:        8,
			StopBits:        1,
			MinimumReadSize: 1,
			ParityMode:      serial.PARITY_NONE,
		}
		o.port, err = serial.Open(options)
		o.initCRC()
		o.stop = false
		o.cmdChannel = make(chan string, 10)
		go processor(o, o.cmdChannel)
		go parser(o)
		go outFrequencyRequester(o)
	})
	return
}

func (o *HyInverter) GCode(cmd string) (ok bool) {
	subCmds := strings.Fields(cmd) // splits by whitespace
	for _, subCmd := range subCmds {
		select {
		case o.cmdChannel <- subCmd:
			ok = true
		default:
			ok = false
		}
	}
	return
}

func processor(handle *HyInverter, commands chan string) {
	for !handle.stop {
		cmd := <-commands
		cmd = strings.TrimSpace(strings.ToLower(cmd))
		if cmd == "end" || cmd == "m0" || cmd == "m1" || cmd == "m30" || cmd == "m60" || cmd == "m5" || cmd == "m05" {
			// Stop
			handle.port.Write(handle.signMessage([]byte{0x01, 0x03, 0x01, 0x08}))
			time.Sleep(time.Millisecond * 110)
		} else if cmd == "m3" || cmd == "m03" {
			// Run Forward
			handle.port.Write(handle.signMessage([]byte{0x01, 0x03, 0x01, 0x01}))
			time.Sleep(time.Millisecond * 110)
		} else if cmd == "m4" || cmd == "m04" {
			// Run Backward
			handle.port.Write(handle.signMessage([]byte{0x01, 0x03, 0x01, 0x11}))
			time.Sleep(time.Millisecond * 110)
		} else if strings.HasPrefix(cmd, "s") {
			outputRpm, err := strconv.ParseUint(cmd[1:], 10, 16)
			if err == nil {
				inverterFrequency := uint16(float32(outputRpm) * inverterHertzPerRpm)
				fBytes := make([]byte, 2)
				binary.BigEndian.PutUint16(fBytes, uint16(inverterFrequency))
				// Set frequency
				handle.port.Write(handle.signMessage([]byte{0x01, 0x05, 0x02, fBytes[0], fBytes[1]}))
				time.Sleep(time.Millisecond * 110)
			} else {
				fmt.Errorf("Could not get freq. out of '%s'\n", cmd)
				fmt.Println(err)
			}
		} else if cmd == "?" {
			// Request current Frequency
			handle.port.Write(handle.signMessage([]byte{0x01, 0x04, 0x03, 0x01, 0x00, 0x00}))
			time.Sleep(time.Millisecond * 110)
		}
	}
}

func outFrequencyRequester(handle *HyInverter) {
	for !handle.stop {
		time.Sleep(time.Millisecond * time.Duration(RequestInverterIntervalMillis))
		handle.GCode("?")
	}
}

func parser(handle *HyInverter) {
	modbusRtu := make([]byte, 0)
	lastRead := time.Now()
	rxBuf := make([]byte, 10)
	for !handle.stop {
		n, err := handle.port.Read(rxBuf)
		read := time.Now()
		if read.Sub(lastRead).Seconds() > 0.05 {
			modbusRtu = make([]byte, 0) // clear buffer if "end" detected
		}
		if n > 0 && err == nil {
			modbusRtu = append(modbusRtu, rxBuf[:n]...)
			parseModbusRTU(handle, modbusRtu)
		}
		lastRead = read
	}
}

func parseModbusRTU(handle *HyInverter, msg []byte) {
	// Request current Frequency
	// 0x01 0x04 0x03 0x01 0x00 0x00 0xA1 0x8E
	if len(msg) == 8 {
		if msg[0] == 0x01 && msg[1] == 0x04 && msg[2] == 0x03 && msg[3] == 0x01 {
			signTest := handle.signMessage(msg[:6])
			if signTest[6] == msg[6] && signTest[7] == msg[7] {
				fBytes := make([]byte, 2)
				fBytes[0] = msg[4]
				fBytes[1] = msg[5]
				handle.outputFrequency = binary.BigEndian.Uint16(fBytes)
				handle.outputRpm = uint16(float32(handle.outputFrequency) / inverterHertzPerRpm)
				handle.lastReceived = time.Now()
			}
		}
	}
}

func (o *HyInverter) OutputFrequency() uint16 {
	return o.outputFrequency
}

func (o *HyInverter) OutputRpm() uint16 {
	return o.outputRpm
}

func (o *HyInverter) Online() bool {
	rxDiff := time.Now().Sub(o.lastReceived)
	if rxDiff.Seconds() < 2 {
		return true
	} else {
		return false
	}
}

func (o *HyInverter) initCRC() {
	o.hash16 = crc16.New(crc16.Modbus)
}

func (o *HyInverter) Close() {
	o.stop = true
	o.port.Close()
}

func (o *HyInverter) signMessage(data []byte) []byte {
	o.hash16.Reset()
	o.hash16.Write(data)
	return o.hash16.Sum(data)
}
