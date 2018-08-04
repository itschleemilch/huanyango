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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// HyInverter is the base of a VFD controller. It contains the configuration and run time variables.
// Example usage:
//
//  handle := &HyInverter{}
//  handle.Open("/dev/ttyUSB0", 11520, 3.47222, 750)
//  defer handle.Close()
//  handle.GCode("M3 S300")
//
type HyInverter struct {
	port            io.ReadWriteCloser
	hash16          crc16.Hash16
	stop            bool
	once            sync.Once
	cmdChannel      chan string
	setFrequency    uint16
	outputFrequency uint16
	outputRpm       uint16
	lastReceived    time.Time
	pollIntervalSec float64
	// The API sets and reads the output frequency, which has a linear relation to output RPM.
	// Experimentally determined: 3.47222 (using the VFD display while spinning)
	rpmToHertz float32
	// Experimentally determined with inverter: 11520 at my setup.
	maxRpm uint16
	// commandQueue is a counter which is increased by the gcode preprocessor and
	// decreased by the gcode interpreter.
	commandQueue int32
}

// gcodeSeparator splits GCODEs missing whitespace.
// Example input: N12S20 F200M3 G28.3Z-100 Y-29.3
// Usage:
//
//   fmt.Println(gcodeSeparator.ReplaceAllString(`N12S20 F200M3 G28.3Z-100 Y-29.3`, `$1 `))
//
// Output: "N12 S20 F200 M3 G28.3 Z-100 Y-29.3 "
var gcodeSeparator *regexp.Regexp = regexp.MustCompile(`([a-zA-Z][\-+]*\d+\.*\d*)\s*`)

// NewVfd creates an empty data struct. Please call Open and defer Close.
func NewVfd() *HyInverter {
	return &HyInverter{}
}

// Open inits a serial port handle and creates all required goroutines.
// Param portName: OS specific refence to a serial port (examples - Windows: COM3, Linux: /dev/ttyUSB0).
// Param maxRpm: Maximum allowed and outputed rpm - for instance 11520 /min.
// Param rpmToHertz: This constant is used to calculate the set frequency for the VFD. If unknown, set
// to 1 and check the VFD display to calculate this value afterwards.
// Param rpmPollInterval: This is used to regularly check the is value of the output frequency.
func (o *HyInverter) Open(portName string, maxRpm uint16, rpmToHertz float64, rpmPollInterval int64) (err error) {
	o.once.Do(func() {
		o.rpmToHertz = float32(rpmToHertz)
		o.maxRpm = maxRpm
		o.pollIntervalSec = float64(rpmPollInterval) / 1000.0
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
		go outFrequencyRequester(o, rpmPollInterval)
	})
	return
}

// GCode is the external control input. It accepts string messages in the standard G-Code format.
// Accepted commands: M2, M3, M4, M5, Sxxx. Aliases for M5: M0, M1, M30, M60.
// Returns true if the command stack has space for the new input.
// This function also acts as a preprocessor since it reformats the input commands.
// Examples:
//
//   M3S400
//   M4 S5000
//   M9 S0 M5
//
func (o *HyInverter) GCode(cmd string) (ok bool) {
	ok = true
	cleanedGcode := gcodeSeparator.ReplaceAllString(cmd, `$1 `)
	subCmds := strings.Fields(cleanedGcode) // splits by whitespace
	atomic.AddInt32(&o.commandQueue, int32(len(subCmds)))
	for _, subCmd := range subCmds {
		select {
		case o.cmdChannel <- subCmd:
			break
		default:
			ok = false
			atomic.AddInt32(&o.commandQueue, -1)
			break
		}
	}
	return
}

func processor(handle *HyInverter, commands chan string) {
	for !handle.stop {
		cmd := <-commands
		atomic.AddInt32(&handle.commandQueue, -1)
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
				inverterFrequency := uint16(float32(outputRpm) * handle.rpmToHertz)
				handle.setFrequency = inverterFrequency
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

func outFrequencyRequester(handle *HyInverter, pollInterval int64) {
	for !handle.stop {
		time.Sleep(time.Millisecond * time.Duration(pollInterval))
		handle.GCode("?")
	}
}

func parser(handle *HyInverter) {
	var modbusRtu []byte = make([]byte, 0)
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
				handle.outputRpm = uint16(float32(handle.outputFrequency) / handle.rpmToHertz)
				handle.lastReceived = time.Now()
			}
		}
	}
}

// OutputFrequency returns the raw value from the VFD.
// Please also check Online() to see if the value is valid.
func (o *HyInverter) OutputFrequency() uint16 {
	return o.outputFrequency
}

// OutputRpm returns the converted output frequency (rpm := output_frequency / rpm-to-hertz).
// Please also check Online() to see if the value is valid.
func (o *HyInverter) OutputRpm() uint16 {
	return o.outputRpm
}

// Online returns true if the last received message by the VFD was lately.
func (o *HyInverter) Online() bool {
	rxDiff := time.Now().Sub(o.lastReceived)
	if rxDiff.Seconds() < 2*o.pollIntervalSec {
		return true
	}
	return false
}

// Processed returns true if all commands were processed and
// the output frequency is within 10% of the set frequency.
func (o *HyInverter) Processed() (processed, outputFrequencyOk, commandsProcessed bool) {
	lowerBound := float32(o.setFrequency) * 0.9
	upperBound := float32(o.setFrequency) * 1.1
	value := float32(o.outputFrequency)
	if value >= lowerBound && value <= upperBound {
		// Range test passed
		outputFrequencyOk = true
	}
	if atomic.LoadInt32(&o.commandQueue) == 0 {
		commandsProcessed = true
	}
	processed = outputFrequencyOk && commandsProcessed
	return
}

func (o *HyInverter) initCRC() {
	o.hash16 = crc16.New(crc16.Modbus)
}

// Close closes all handles and goroutines.
func (o *HyInverter) Close() {
	o.stop = true
	o.port.Close()
}

func (o *HyInverter) signMessage(data []byte) []byte {
	o.hash16.Reset()
	o.hash16.Write(data)
	return o.hash16.Sum(data)
}
