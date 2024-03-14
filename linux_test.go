//go:build linux
// +build linux

package mbserver

import (
	"log"
	"os/exec"
	"testing"
	"time"

	"github.com/goburrow/modbus"
	"go.bug.st/serial"
)

//TODO socat stream not reading

// The serial read and close has a known race condition.
// https://github.com/golang/go/issues/10001
func TestModbusRTU(t *testing.T) {

	var nameSerial1, nameSerial2 string = "/tmp/ttyFOO", "/tmp/ttyBAR"
	var err error

	// Create a pair of virtual serial devices.
	cmd := exec.Command("socat",
		"pty,raw,echo=0,link="+nameSerial1,
		"pty,raw,echo=0,link="+nameSerial2,
	)
	err = cmd.Start()
	if err != nil {
		log.Fatal("socat not start ", err)
	}

	defer cmd.Wait()
	defer cmd.Process.Kill()

	// Allow the virtual serial devices to be created.
	time.Sleep(100 * time.Millisecond)

	// Server
	var LowerID, UpperID byte = 1, 1
	s := NewServer(LowerID, UpperID, 30000, 30000)
	err = s.ListenRTU(nameSerial1,
		&serial.Mode{BaudRate: 115200,
			DataBits:          8,
			Parity:            serial.NoParity,
			StopBits:          serial.OneStopBit,
			InitialStatusBits: nil,
		},
	)
	if err != nil {
		t.Fatalf("failed to listen, got %v\n", err)
	}
	// after change library serial and use socat not work meth Close
	defer s.Close()

	// Allow the server to start and to avoid a connection refused on the client
	time.Sleep(10 * time.Millisecond)
	// Client
	handler := modbus.NewRTUClientHandler(nameSerial2)
	handler.BaudRate = 115200
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = 1
	handler.Timeout = 5 * time.Second
	// Connect manually so that multiple requests are handled in one connection session
	err = handler.Connect()
	if err != nil {
		t.Errorf("failed to connect, got %v\n", err)
		t.FailNow()
	}
	defer func() {
		err := handler.Close()
		if err != nil {
			log.Println(err)
		}

	}()

	client := modbus.NewClient(handler)

	// Coils
	_, err = client.WriteMultipleCoils(100, 9, []byte{255, 1})
	if err != nil {
		t.Errorf("expected nil, got %v\n", err)
		t.FailNow()
	}

	results, err := client.ReadCoils(100, 16)
	if err != nil {
		t.Errorf("expected nil, got %v\n", err)
		t.FailNow()
	}

	expect := []byte{255, 1}
	got := results
	if !isEqual(expect, got) {
		t.Errorf("expected %v, got %v", expect, got)
	}
}
