package mbserver

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"strings"
)

func (s *Server) accept(listen net.Listener) error {
	for {
		conn, err := listen.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}
			log.Printf("Unable to accept connections: %#v\n", err)
			return err
		}

		go func(conn net.Conn) {
			defer conn.Close()

			for {
				// some devices may send multiple requests in one packet
				// we need reassebmly frame bytes in two steps 
				
				// step 1 - read first 6 bytes 
				frameStart := make([]byte, 6)
				
				_, err := conn.Read(frameStart)
				if err != nil { 
					if err != io.EOF { 
						log.Printf("read frameStart error %v\n", err)
					}
					return
				}

				// step 2 - read frameEnd with length specified in 6th byte of frameStart 
				frameEnd := make([]byte, int(frameStart[5])) 
				_, err = conn.Read(frameEnd)
				if err != nil { 
					if err != io.EOF { 
						log.Printf("read frameEnd error %v\n", err)
					}
					return
				}

				// Set the length of the packet to the number of read bytes.
				packet := append(frameStart, frameEnd...)
				frame, err := NewTCPFrame(packet)
				if err != nil {
					log.Printf("bad packet error %v\n", err)
					log.Printf("frame data: % x\n", packet)
					return
				}

				request := &Request{conn, frame}

				s.requestChan <- request
			}
		}(conn)
	}
}

// ListenTCP starts the Modbus server listening on "address:port".
func (s *Server) ListenTCP(addressPort string) (err error) {
	listen, err := net.Listen("tcp", addressPort)
	if err != nil {
		log.Printf("Failed to Listen: %v\n", err)
		return err
	}
	s.listeners = append(s.listeners, listen)
	go s.accept(listen)
	return err
}

// ListenTLS starts the Modbus server listening on "address:port".
func (s *Server) ListenTLS(addressPort string, config *tls.Config) (err error) {
	listen, err := tls.Listen("tcp", addressPort, config)
	if err != nil {
		log.Printf("Failed to Listen on TLS: %v\n", err)
		return err
	}
	s.listeners = append(s.listeners, listen)
	go s.accept(listen)
	return err
}
