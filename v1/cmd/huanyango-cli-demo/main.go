// Copyright (c) 2018 Sebastian Schleemilch
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"ksgbr/cnc6040/hyinverter"
	"os"
)

func main() {

	fmt.Println("HyInverter Demo App")

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter serial port: ")
	scanner.Scan()

	portName := scanner.Text()

	fmt.Println()
	fmt.Printf("Selected Port: %s\n", portName)

	hyInv := &hyinverter.HyInverter{}
	err := hyInv.Open(portName)
	defer hyInv.Close()
	if err != nil {
		panic(err)
	}
	continueScanning := true
	for continueScanning && scanner.Scan() {
		cmd := scanner.Text()
		if cmd == "?" {
			fmt.Println("Output RPM 1/min: ", hyInv.OutputRpm())
		} else if cmd == "help" {
			fmt.Println("Commands: M3 <freq>, M4 <freq>, M5, ?")
		} else if cmd == "$" {
			fmt.Println("HyInverter Connected and Online: ", hyInv.Online())
		} else if cmd == "." {
			continueScanning = false
		} else {
			hyInv.GCode(cmd)
		}
	}
	fmt.Println("End.")
}
