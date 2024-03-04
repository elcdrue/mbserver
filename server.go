// Package mbserver implments a Modbus server (slave).
package mbserver

import (
	"go.bug.st/serial"
	"io"
	"net"
	"sync"
)

// Server is a Modbus slave with allocated memory for discrete inputs, coils, etc.
type Server struct {
	// Debug enables more verbose messaging.
	Debug                bool
	listeners            []net.Listener
	ports                []serial.Port
	portsWG              sync.WaitGroup
	portsCloseChan       chan struct{}
	requestChan          chan *Request
	function             [256](func(*Server, Framer) ([]byte, *Exception))
	slaves               []SlaveMemory
	lowerSlaveId         byte
	upperSlaveId         byte
	offsetInputRegisters uint16 // offset to copy from HR to IR
	offsetDiscreteInputs uint16 // offset to copy from Coils to DI
	ListenState
}

type ListenState struct {
	state     int
	bytesLeft int
	buffer    []byte
	packet    []byte
	hasErr    bool
}

type SlaveMemory struct {
	DiscreteInputs   []byte
	Coils            []byte
	HoldingRegisters []uint16
	InputRegisters   []uint16
}

// Request contains the connection and Modbus frame.
type Request struct {
	conn  io.ReadWriteCloser
	frame Framer
}

// NewServer creates a new Modbus server (slave).
func NewServer(LowerID, UpperID byte, OffsetInputRegisters uint16, OffsetDiscreteInputs uint16) *Server {
	var i byte
	s := &Server{}
	s.lowerSlaveId = LowerID
	s.upperSlaveId = UpperID
	s.offsetInputRegisters = OffsetInputRegisters
	s.offsetDiscreteInputs = OffsetDiscreteInputs

	length := s.upperSlaveId - s.lowerSlaveId + 1
	slaves := make([]SlaveMemory, length)

	// Allocate Modbus memory maps.
	for i = 0; i < length; i++ {
		slaves[i].DiscreteInputs = make([]byte, 65536)
		slaves[i].Coils = make([]byte, 65536)
		slaves[i].HoldingRegisters = make([]uint16, 65536)
		slaves[i].InputRegisters = make([]uint16, 65536)
	}

	s.slaves = slaves

	// Add default functions.
	s.function[1] = ReadCoils
	s.function[2] = ReadDiscreteInputs
	s.function[3] = ReadHoldingRegisters
	s.function[4] = ReadInputRegisters
	s.function[5] = WriteSingleCoil
	s.function[6] = WriteHoldingRegister
	s.function[15] = WriteMultipleCoils
	s.function[16] = WriteHoldingRegisters

	ls := ListenState{}
	ls.buffer = make([]byte, 256)
	ls.hasErr = false
	ls.bytesLeft = 0
	s.ListenState = ls

	s.requestChan = make(chan *Request)
	s.portsCloseChan = make(chan struct{})

	go s.handler()

	return s
}

// RegisterFunctionHandler override the default behavior for a given Modbus function.
func (s *Server) RegisterFunctionHandler(funcCode uint8, function func(*Server, Framer) ([]byte, *Exception)) {
	s.function[funcCode] = function
}

func (s *Server) handle(request *Request) Framer {
	var exception *Exception
	var data []byte

	slaveId := request.frame.GetAddress()
	response := request.frame.Copy()
	function := request.frame.GetFunction()

	if slaveId < s.lowerSlaveId || slaveId > s.upperSlaveId {
		return nil
	} else if s.function[function] != nil {
		data, exception = s.function[function](s, request.frame)
		response.SetData(data)
	} else {
		exception = &IllegalFunction
	}

	if exception != &Success {
		response.SetException(exception)
	}

	return response
}

// All requests are handled synchronously to prevent modbus memory corruption.
func (s *Server) handler() {
	for {
		request := <-s.requestChan
		response := s.handle(request)
		if response != nil {
			request.conn.Write(response.Bytes())
		}
	}
}

// Close stops listening to TCP/IP ports and closes serial ports.
func (s *Server) Close() {
	for _, listen := range s.listeners {
		listen.Close()
	}

	close(s.portsCloseChan)
	s.portsWG.Wait()

	for _, port := range s.ports {
		port.Close()
	}
}
