package mbserver

import (
	"go.bug.st/serial"
	"log"
	"time"
)

// ListenRTU starts the Modbus server listening to a serial device.
// For example:  err := s.ListenRTU(&serial.Config{Address: "/dev/ttyUSB0"})
func (s *Server) ListenRTU(name string, mode *serial.Mode) (err error) {
	port, err := serial.Open(name, mode)
	if err != nil {
		log.Fatalf("failed to open %s: %v\n", name, err)
	}

	err = port.SetMode(mode)
	if err != nil {
		log.Print(err)
	}

	err = port.SetReadTimeout(5 * time.Millisecond)
	if err != nil {
		log.Print(err)
	}

	buffer = make([]byte, 256)

	s.ports = append(s.ports, port)

	s.portsWG.Add(1)
	go func() {
		defer s.portsWG.Done()
		s.acceptSerialRequests(port)
	}()

	return err
}

// TODO: think about use multiple serial ports in one application
var buffer []byte

func (s *Server) acceptSerialRequests(port serial.Port) {
	const (
		InitialState = iota // receive all incoming data from serial port on init or err
		ReceiveState        // try to read byte stream from port before timeout
		ControlState        // return of reconstructed frame to check data and prepare response
	)

	var bytesRead int
	var hasReceivedData bool
	var err error

	for {
		select {
		case <-s.portsCloseChan:
			return
		default:
		}

		switch s.ListenState.state {

		case InitialState:
			s.ListenState.hasErr = false
			bytesRead, err = port.Read(buffer)

			if err != nil {
				log.Print("InitialState err", err)
			}

			if bytesRead == 0 {
				hasReceivedData = false
				s.ListenState.state = ReceiveState
				continue
			}

		case ReceiveState:

			if !hasReceivedData {
				s.ListenState.packet = []byte{}
			}

			bytesRead, err := port.Read(buffer)
			if err != nil {
				log.Print("Receive state err", err)
			}

			// go to check received data
			if bytesRead == 0 && hasReceivedData {
				s.ListenState.state = ControlState

				// append received data to buffer
			} else if bytesRead > 0 {
				hasReceivedData = true
				s.ListenState.packet = append(s.ListenState.packet, buffer[0:bytesRead]...)
			}

		case ControlState:

			hasReceivedData = false
			// check frame and build response
			frame, err := NewRTUFrame(s.ListenState.packet)
			if err != nil {
				s.ListenState.state = InitialState
				continue
			}
			s.ListenState.state = ReceiveState
			hasReceivedData = false
			// write request to the channel
			request := &Request{port, frame}
			s.requestChan <- request

		}
	}
}
