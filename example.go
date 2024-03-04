package mbserver

/*

package main

import (
	"flag"
	"github.com/elcdrue/mbserver"
	"go.bug.st/serial"
	"log"
	"strconv"
	"time"
)

var iLowerID, iUpperID, iTcpPort, iComPort, iBaudRate, iDataBits, iStopBits, iParity, iTimeOut int
var sIp string

func main() {
	flag.StringVar(&sIp, "ip", "0.0.0.0", "listen on ip")
	flag.IntVar(&iLowerID, "lo", 1, "Lower Slave Unit ID")
	flag.IntVar(&iUpperID, "up", 1, "Upper Slave Unit ID")
	flag.IntVar(&iTcpPort, "port", 1502, "Listen on TCP port num")
	flag.IntVar(&iComPort, "com", 0, "Listen on Com port num")
	flag.IntVar(&iBaudRate, "speed", 19200, "Baudrate of com port")
	flag.IntVar(&iDataBits, "databits", 8, "Databits of com port")
	flag.IntVar(&iStopBits, "stopbits", 1, "stopbits of com port")
	flag.IntVar(&iParity, "parity", 0, "Parity, 0=none, 1=odd, 2=even")
	flag.IntVar(&OffsetInput, "ofs", 10000, "Offset of holding register copy it to input")
	flag.Parse()

	var lowerID byte = byte(iLowerID)
	var upperID byte = byte(iUpperID)

	serv := mbserver.NewServer(lowerID, upperID)
	tcpPort := strconv.Itoa(iTcpPort)

	if iComPort != 444 {

		comName := "/dev/ttyUSB" + strconv.Itoa(iComPort)
		mode := &serial.Mode{
			BaudRate: iBaudRate,
			DataBits: iDataBits,
			Parity:   setParity(iParity),
			StopBits: setStopBits(iStopBits),
		}

		err := serv.ListenRTU(comName, mode)
		if err != nil {
			log.Fatalf("failed to listen, got %v\n", err)
		}
	}

	err := serv.ListenTCP("0.0.0.0:" + tcpPort)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	defer serv.Close()

	// Wait forever
	for {
		time.Sleep(1 * time.Second)
	}
}

func setParity(parity int) (serialParity serial.Parity) {

	switch parity {
	case 0:
		serialParity = serial.NoParity
	case 1:
		serialParity = serial.OddParity
	case 2:
		serialParity = serial.EvenParity
	default:
		serialParity = serial.NoParity
	}
	return
}

func setStopBits(stopBits int) (serialStopBits serial.StopBits) {

	switch stopBits {
	case 1:
		serialStopBits = serial.OneStopBit
	case 2:
		serialStopBits = serial.TwoStopBits
	case 15:
		serialStopBits = serial.OnePointFiveStopBits
	}
	return
}


*/
