// Copyright (c) 2018 Sebastian Schleemilch
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package vfdio

import "testing"

func TestModbusCrc16(t *testing.T) {
	hy := &HyInverter{}
	hy.initCRC()
	msg := hy.signMessage([]byte{0x01, 0x03, 0x01, 0x08})
	if len(msg) != 6 {
		t.FailNow()
	}
	if msg[4] != 0xF1 {
		t.FailNow()
	}
	if msg[5] != 0x8E {
		t.FailNow()
	}
}
