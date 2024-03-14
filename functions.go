package mbserver

import (
	"encoding/binary"
	"log"
)

// ReadCoils function 1, reads coils from internal memory.
func ReadCoils(s *Server, frame Framer) ([]byte, *Exception) {
	register, numRegs, endRegister := registerAddressAndNumber(frame)

	if (int(register) + int(numRegs)) > 65536 {
		return []byte{}, &IllegalDataAddress
	}
	dataSize := numRegs / 8
	if (numRegs % 8) != 0 {
		dataSize++
	}
	data := make([]byte, 1+dataSize)
	data[0] = byte(dataSize)
	slaveID := frame.GetAddress()
	idx := s.upperSlaveId - slaveID
	for i, value := range s.slaves[idx].Coils[register:endRegister] {
		if value != 0 {
			shift := uint(i) % 8
			data[1+i/8] |= byte(1 << shift)
		}
	}
	return data, &Success
}

// ReadDiscreteInputs function 2, reads discrete inputs from internal memory.
func ReadDiscreteInputs(s *Server, frame Framer) ([]byte, *Exception) {
	register, numRegs, endRegister := registerAddressAndNumber(frame)

	if (int(register) + int(numRegs)) > 65536 {
		return []byte{}, &IllegalDataAddress
	}
	dataSize := numRegs / 8
	if (numRegs % 8) != 0 {
		dataSize++
	}
	data := make([]byte, 1+dataSize)
	data[0] = byte(dataSize)
	slaveID := frame.GetAddress()
	idx := s.upperSlaveId - slaveID
	for i, value := range s.slaves[idx].DiscreteInputs[register:endRegister] {
		if value != 0 {
			shift := uint(i) % 8
			data[1+i/8] |= byte(1 << shift)
		}
	}
	return data, &Success
}

// ReadHoldingRegisters function 3, reads holding registers from internal memory.
func ReadHoldingRegisters(s *Server, frame Framer) ([]byte, *Exception) {
	register, numRegs, endRegister := registerAddressAndNumber(frame)
	if (int(register) + int(numRegs)) > 65536 {
		return []byte{}, &IllegalDataAddress
	}
	slaveID := frame.GetAddress()
	idx := s.upperSlaveId - slaveID
	return append([]byte{byte(numRegs * 2)}, Uint16ToBytes(s.slaves[idx].HoldingRegisters[register:endRegister])...), &Success
}

// ReadInputRegisters function 4, reads input registers from internal memory.
func ReadInputRegisters(s *Server, frame Framer) ([]byte, *Exception) {
	register, numRegs, endRegister := registerAddressAndNumber(frame)
	if (int(register) + int(numRegs)) > 65536 {
		return []byte{}, &IllegalDataAddress
	}
	slaveID := frame.GetAddress()
	idx := s.upperSlaveId - slaveID
	return append([]byte{byte(numRegs * 2)}, Uint16ToBytes(s.slaves[idx].InputRegisters[register:endRegister])...), &Success
}

// WriteSingleCoil function 5, write a coil to internal memory.
func WriteSingleCoil(s *Server, frame Framer) ([]byte, *Exception) {
	register, value := registerAddressAndValue(frame)
	// TODO Should we use 0 for off and 65,280 (FF00 in hexadecimal) for on?
	if value != 0 {
		value = 1
	}
	slaveID := frame.GetAddress()
	idx := s.upperSlaveId - slaveID
	s.slaves[idx].Coils[register] = byte(value)
	// copy to di from coils with offset
	if register >= s.offsetDiscreteInputs {
		s.slaves[idx].DiscreteInputs[register-s.offsetDiscreteInputs] = byte(value)
	}

	return frame.GetData()[0:4], &Success
}

// WriteHoldingRegister function 6, write a holding register to internal memory.
func WriteHoldingRegister(s *Server, frame Framer) ([]byte, *Exception) {
	register, value := registerAddressAndValue(frame)
	slaveID := frame.GetAddress()
	idx := s.upperSlaveId - slaveID
	s.slaves[idx].HoldingRegisters[register] = value
	// copy value from holding register with offset
	if uint16(register) >= s.offsetInputRegisters {
		s.slaves[idx].InputRegisters[register-s.offsetInputRegisters] = value
	}

	return frame.GetData()[0:4], &Success
}

// WriteMultipleCoils function 15, writes holding registers to internal memory.
func WriteMultipleCoils(s *Server, frame Framer) ([]byte, *Exception) {
	register, numRegs, _ := registerAddressAndNumber(frame)
	valueBytes := frame.GetData()[5:]

	if (int(register) + int(numRegs)) > 65536 {
		return []byte{}, &IllegalDataAddress
	}

	// TODO This is not correct, bits and bytes do not always align
	//if len(valueBytes)/2 != numRegs {
	//	return []byte{}, &IllegalDataAddress
	//}
	slaveID := frame.GetAddress()
	idx := s.upperSlaveId - slaveID
	var bitCount uint16 = 0
	for i, value := range valueBytes {
		for bitPos := uint16(0); bitPos < 8; bitPos++ {
			s.slaves[idx].Coils[register+uint16(i*8)+bitPos] = bitAtPosition(value, bitPos)
			if s.offsetDiscreteInputs <= register+uint16(i*8)+bitPos {
				s.slaves[idx].DiscreteInputs[register+uint16(i*8)+bitPos-s.offsetDiscreteInputs] = bitAtPosition(value, bitPos)
			}
			bitCount++
			if bitCount >= numRegs {
				break
			}
		}
		if bitCount >= numRegs {
			break
		}
	}

	return frame.GetData()[0:4], &Success
}

// WriteHoldingRegisters function 16, writes holding registers to internal memory.
func WriteHoldingRegisters(s *Server, frame Framer) ([]byte, *Exception) {
	register, numRegs, _ := registerAddressAndNumber(frame)
	valueBytes := frame.GetData()[5:]
	var exception *Exception
	var data []byte

	if uint16(len(valueBytes)/2) != numRegs || (int(register)+int(numRegs)) > 65535 {
		exception = &IllegalDataAddress
	}
	slaveID := frame.GetAddress()
	idx := s.upperSlaveId - slaveID
	// Copy data to memory
	values := BytesToUint16(valueBytes)
	valuesUpdated := copy(s.slaves[idx].HoldingRegisters[register:], values)
	// copy value from holding register with offset
	if register >= s.offsetInputRegisters && exception != &IllegalDataAddress {
		valuesUpdated1 := copy(s.slaves[idx].InputRegisters[register-s.offsetInputRegisters:], values)
		if valuesUpdated != valuesUpdated1 {
			log.Println("not succesfully copied holding register to input registers")
		}
	}

	if uint16(valuesUpdated) == numRegs {
		exception = &Success
		data = frame.GetData()[0:4]
	} else {
		exception = &IllegalDataAddress
	}

	return data, exception
}

// BytesToUint16 converts a big endian array of bytes to an array of unit16s
func BytesToUint16(bytes []byte) []uint16 {
	values := make([]uint16, len(bytes)/2)

	for i := range values {
		values[i] = binary.BigEndian.Uint16(bytes[i*2 : (i+1)*2])
	}
	return values
}

// Uint16ToBytes converts an array of uint16s to a big endian array of bytes
func Uint16ToBytes(values []uint16) []byte {
	bytes := make([]byte, len(values)*2)

	for i, value := range values {
		binary.BigEndian.PutUint16(bytes[i*2:(i+1)*2], value)
	}
	return bytes
}

func bitAtPosition(value uint8, pos uint16) uint8 {
	return (value >> pos) & 0x01
}
