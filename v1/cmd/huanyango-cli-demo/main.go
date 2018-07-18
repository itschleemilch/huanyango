// Copyright (c) 2018 Sebastian Schleemilch
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

// This is a demo app that uses the Huanyango library to control a Huanyang VFD.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/itschleemilch/huanyango/v1/vfdio"
	"os"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "huanyango-cli-demo -port=/dev/ttyUSB0")
		fmt.Fprintln(flag.CommandLine.Output())
		fmt.Fprintln(flag.CommandLine.Output(), "Use G-Codes M3, M4, M4 and Snnnn.")
		fmt.Fprintln(flag.CommandLine.Output(), "? prints the current RPM.")
		fmt.Fprintln(flag.CommandLine.Output(), "$ outputs if connected.")
		fmt.Fprintln(flag.CommandLine.Output())
		fmt.Fprintln(flag.CommandLine.Output())
		flag.PrintDefaults()
	}
	var serialDevice *string = flag.String("port", "/dev/ttyUSB0", "USB Port. Linux default: /dev/ttyUSB0. On Windows use COMx, e.g. COM3.")
	var pollRate *int64 = flag.Int64("interval", 750, "RPM status readout interval in milliseconds. Default: 750.")
	var rpmHertzConversation *float64 = flag.Float64("rpm2hz", 3.47222, "Unit conversation from RPM to Hz. May be determined experimentally.")
	var maxRpm *int64 = flag.Int64("maxrpm", 11520, "Maximum allowed RPM for your spindle.")
	flag.Parse()

	fmt.Println("Huanyango Command Line Interface Demo")
	fmt.Println("Commands: M3, M4, M5, Snnnn, ?, $, exit, help")

	hyInv := vfdio.NewVfd()
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Failed to open serial port '", *serialDevice, "'. Use --help flag.")
		}
	}()
	err := hyInv.Open(*serialDevice, uint16(*maxRpm), *rpmHertzConversation, *pollRate)
	defer hyInv.Close()
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(os.Stdin)
	continueScanning := true
	fmt.Print("> ")
	for continueScanning && scanner.Scan() {
		cmd := scanner.Text()
		if cmd == "?" {
			fmt.Println("Output RPM 1/min: ", hyInv.OutputRpm())
		} else if cmd == "help" {
			fmt.Println("Commands: M3, M4, M5, Snnnn, $, ?, exit, help.")
		} else if cmd == "$" {
			fmt.Println("Commands: M3, M4, M5, Snnnn, ?, $, exit, help")
		} else if cmd == "exit" {
			continueScanning = false
			break
		} else {
			hyInv.GCode(cmd)
		}
		fmt.Print("> ")
	}
	fmt.Println("End.")
}
