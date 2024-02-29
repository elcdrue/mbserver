package mbserver

import (
	"go.bug.st/serial"
	"io"
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

	err = port.SetReadTimeout(1 * time.Millisecond)
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
		SERIAL_RECV_INIT  = iota // flush all incoming data from serial port on init or err
		SERIAL_RECV_START        // try to read 8 start bytes of frame
		SERIAL_RECV_END          // if frame length greater than 8 bytes try to read more bytes
		SERIAL_RECV_RET          // return of reconstructed frame to check data and prepare response
	)

	var bytesRead int
	var err error

	for {
		select {
		case <-s.portsCloseChan:
			return
		default:
		}

		switch s.ListenState.state {

		case SERIAL_RECV_INIT:

			s.ListenState.hasErr = false
			bytesRead, err = port.Read(buffer)

			if err != nil {
				log.Print("SERIAL_RECV_INIT err", err)
			}

			if bytesRead == 0 {
				s.ListenState.state = SERIAL_RECV_START
				continue
			}

		case SERIAL_RECV_START:

			s.ListenState.bytesLeft = 0
			s.ListenState.packet = []byte{}
			bytesRead, err := io.ReadFull(port, buffer[0:8])
			if err != nil {
				log.Print("SERIAL_RECV_START err", err)
			}

			if bytesRead == 8 {

				s.ListenState.packet = append(s.ListenState.packet, buffer[0:bytesRead]...)
				switch {
				/*	slave id not specified,
					maybe need implement broadcast requests? */
				case buffer[00] == 0:
					s.ListenState.state = SERIAL_RECV_INIT

				/* request length is 8 bytes
				buffer[00] = slaveId
				buffer[01] = function code
				buffer[02] = start register hi
				buffer[03] = start register lo
				buffer[04] = size of registers hi
				buffer[05] = size of registers lo
				buffer[06] = crc hi
				buffer[07] = crc lo
				*/

				case buffer[01] == 0x01, buffer[01] == 0x02, buffer[01] == 0x03, buffer[01] == 0x04,
					buffer[01] == 0x05, buffer[01] == 0x06:

					s.ListenState.bytesLeft = 0
					s.ListenState.state = SERIAL_RECV_RET

				/* request length is greater than 8 bytes up to 256 bytes
				buffer[00] = slaveId
				buffer[01] = function code
				buffer[02] = start register hi
				buffer[03] = start register lo
				buffer[04] = size of registers hi
				buffer[05] = size of registers lo
				buffer[06] = size of bytes data (x)
				buffer[06..06+x] = size of bytes data
				buffer[06+x+1] = crc hi
				buffer[06+x+2] = crc lo
				*/
				case buffer[01] == 0x0F, buffer[01] == 0x10:
					s.ListenState.bytesLeft = int(buffer[06]) + 1
					s.ListenState.state = SERIAL_RECV_END

				// got err.. non-implemented function?
				default:
					s.ListenState.state = SERIAL_RECV_INIT
				}
			}

		case SERIAL_RECV_END:

			bytesRead, err = port.Read(s.ListenState.buffer)
			if err != nil {
				log.Print("SERIAL_RECV_END error", err)
			}

			// read more bytes
			if bytesRead > 0 {
				// but not full request
				if s.ListenState.bytesLeft > bytesRead {
					s.ListenState.packet = append(s.ListenState.packet, s.ListenState.buffer[:bytesRead]...)
					s.ListenState.bytesLeft -= bytesRead
					// read full request
				} else {

					s.ListenState.packet = append(s.ListenState.packet, s.ListenState.buffer[:s.ListenState.bytesLeft]...)
					s.ListenState.state = SERIAL_RECV_RET

					// if read more bytes than length of frame
					//  serial port is need flush incoming data
					s.ListenState.hasErr = bytesRead > s.ListenState.bytesLeft
				}
			}
		case SERIAL_RECV_RET:
			if s.ListenState.hasErr {
				s.ListenState.state = SERIAL_RECV_INIT
			} else {
				s.ListenState.state = SERIAL_RECV_START
			}

			// check frame and build response
			frame, err := NewRTUFrame(s.ListenState.packet)
			if err != nil {
				log.Printf("bad serial frame error %v\n", err)
				//The next line prevents RTU server from exiting when it receives a bad frame. Simply discard the erroneous
				//frame and wait for next frame by jumping back to the beginning of the 'for' loop.
				log.Printf("Keep the RTU server running!!\n")
				return
			}

			// write request to the channel
			request := &Request{port, frame}
			s.requestChan <- request

		}
	}
}
