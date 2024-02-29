package mbserver

import (
	"io"
	"log"

	"github.com/goburrow/serial"
)

// ListenRTU starts the Modbus server listening to a serial device.
// For example:  err := s.ListenRTU(&serial.Config{Address: "/dev/ttyUSB0"})
func (s *Server) ListenRTU(serialConfig *serial.Config) (err error) {
	port, err := serial.Open(serialConfig)
	if err != nil {
		log.Fatalf("failed to open %s: %v\n", serialConfig.Address, err)
	}
	s.ports = append(s.ports, port)

	s.portsWG.Add(1)
	go func() {
		defer s.portsWG.Done()
		s.acceptSerialRequests(port)
	}()

	return err
}

func (s *Server) acceptSerialRequests(port serial.Port) {
	const (
		INIT   = iota // инициализация
		START         // приём начала пакета
		END           // приём оставшейся части пакета
		RETURN        // возврат собранного пакета

	)

SkipFrameError:

	// проверка на закрытие порта
	for {
		select {
		case <-s.portsCloseChan:
			return
		default:
		}

		switch s.ListenState.state {

		case INIT:
			s.ListenState.hasErr = false
			bytesRead, _ := port.Read(s.ListenState.buffer)
			if bytesRead == 0 {
				break
			}

		case START:
			s.ListenState.bytesLeft = 0
			s.ListenState.packet = []byte{}
			bytesRead, _ := io.ReadFull(port, s.ListenState.buffer[0:8])

			if bytesRead > 0 {
				s.ListenState.packet = append(s.ListenState.packet, s.ListenState.buffer[0:bytesRead]...)

				switch {

				case s.ListenState.packet[00] == 0:
					s.ListenState.state = INIT

				case s.ListenState.packet[01] == 0x01,
					s.ListenState.packet[01] == 0x02,
					s.ListenState.packet[01] == 0x03,
					s.ListenState.packet[01] == 0x04,
					s.ListenState.packet[01] == 0x05,
					s.ListenState.packet[01] == 0x06:

					s.ListenState.bytesLeft = 8 - bytesRead
					s.ListenState.state = RETURN

				case s.ListenState.packet[01] == 0x0F,
					s.ListenState.packet[01] == 0x10:
					s.ListenState.bytesLeft = int(s.ListenState.packet[06]) + 1
					s.ListenState.state = END

				default:
					s.ListenState.state = INIT
				}

			}

		case END:
			bytesRead := 0
			bytesRead, _ = port.Read(s.ListenState.buffer)

			if bytesRead > 0 { // если чтото прочитали

				if s.ListenState.bytesLeft > bytesRead { // прочитали не всё
					s.ListenState.packet = append(s.ListenState.packet, s.ListenState.buffer[:bytesRead]...)
					s.ListenState.bytesLeft -= bytesRead

				} else { // прочитали всё или даже больше
					s.ListenState.hasErr = bytesRead > s.ListenState.bytesLeft
					s.ListenState.packet = append(s.ListenState.packet, s.ListenState.buffer[:s.ListenState.bytesLeft]...)
					s.ListenState.state = RETURN
				}
			}
		case RETURN:
			if s.ListenState.hasErr {
				s.ListenState.state = INIT
			} else {
				s.ListenState.state = START
			}

			frame, err := NewRTUFrame(s.ListenState.packet)
			if err != nil {
				log.Printf("bad serial frame error %v\n", err)
				//The next line prevents RTU server from exiting when it receives a bad frame. Simply discard the erroneous
				//frame and wait for next frame by jumping back to the beginning of the 'for' loop.
				log.Printf("Keep the RTU server running!!\n")
				continue SkipFrameError
				//return
			}

			request := &Request{port, frame}
			s.requestChan <- request

		}
	}
}

/*
	buffer = make([]byte, 256)

			bytesRead, err := port.Read(buffer)
			if err != nil {
				if err != io.EOF {
					log.Printf("serial read error %v\n", err)
				}
				return
			}

			if bytesRead != 0 {

				// Set the length of the packet to the number of read bytes.
				packet := buffer[:bytesRead]

				frame, err := NewRTUFrame(packet)
				if err != nil {
					log.Printf("bad serial frame error %v\n", err)
					//The next line prevents RTU server from exiting when it receives a bad frame. Simply discard the erroneous
					//frame and wait for next frame by jumping back to the beginning of the 'for' loop.
					log.Printf("Keep the RTU server running!!\n")

					//return
				}

				request := &Request{port, frame}

				s.requestChan <- request
			}
		}
*/
