// mb1984 project main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/elcdrue/mbserver"
	"go.bug.st/serial"
)

var iLowerID, iUpperID, iTcpPort, iBaudRate, iDataBits, iStopBits, iParity, iTimeOut int
var ofs1, ofs2 int
var sComPort, sIp, sFunction string

func isValidIP(sIp string) {
	ip := net.ParseIP(sIp)
	return ip != nil
}

func listComPorts() {

}

func main() {
	flag.StringVar(&sIp, "ip", "0.0.0.0", "Listen on IP")
	flag.IntVar(&iLowerID, "lo", 1, "Lower Slave Unit ID")
	flag.IntVar(&iUpperID, "up", 1, "Upper Slave Unit ID")
	flag.IntVar(&iTcpPort, "port", 502, "Listen on TCP port")
	flag.StringVar(&sComPort, "com", "", "Listen on COM port")
	flag.IntVar(&iBaudRate, "speed", 19200, "Baudrate of COM port")
	flag.IntVar(&iDataBits, "data", 8, "Databits of COM port")
	flag.IntVar(&iStopBits, "stop", 1, "Stopbits of COM port")
	flag.IntVar(&iParity, "parity", 0, "Parity, 0=none, 1=odd, 2=even")
	flag.IntVar(&ofs1, "ofs1", 30000, "Offset to copy Holding registers to Input Registers")
	flag.IntVar(&ofs2, "ofs2", 30000, "Offset to copy Coils to Discrete Inputs")

	flag.Parse()

	var lowerID byte = byte(iLowerID)
	var upperID byte = byte(iUpperID)

	if lowerID == 0 {
		log.Fatalln("Incorrect lower Unit ID!")
	}

	if upperID < lowerID || upperID == 0 {
		log.Fatalln("Incorrent upper Unit ID!")
		upperID = lowerID
	}

	if iTcpPort < 1 || iTcpPort > 65535 {
		log.Fatalln("Incorrect TCP port!")
	}

	if !isValidIP(sIp) {
		log.Fatalln("Incorrent IP address!")
	}

	serv := mbserver.NewServer(lowerID, upperID, uint16(ofs1), uint16(ofs2))
	tcpPort := strconv.Itoa(iTcpPort)

	if sComPort == "list" {
		ports, err := serial.GetPortsList()
		if err != nil {
			log.Fatal(err)
		}

		if len(ports) == 0 {
			log.Fatal("No serial ports found!")
		}

		for _, port := range ports {
			fmt.Printf("Found port: %v\n", port)
		}

		return
	} else if sComPort != "" {
		log.Println("use com port with name", sComPort)

		comName := sComPort
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

	log.Printf("===================================\n")
	if iLowerID < iUpperID {
		log.Printf("Server will respond to requests with SlaveID from %d to %d\n", iLowerID, iUpperID)
	} else {
		log.Printf("Server will respond to requests with SlaveID %d\n", iLowerID)
	}

	log.Printf("===================================\n")
	log.Printf("TCP Server listen on ip: %s port: %d\n", sIp, iTcpPort)
	log.Printf("===================================\n")
	if sComPort != "" {
		log.Printf("COM port parameters:\n")
		log.Printf("Name: %s\n", sComPort)
		log.Printf("Speed: %d\n", iBaudRate)
		log.Printf("Data bits: %d\n", iDataBits)
		log.Printf("Stop bits: %d\n", iStopBits)
		if iParity == 0 {
			log.Println("Parity: None")
		}
		if iParity == 1 {
			log.Println("Parity: Odd")
		}
		if iParity == 2 {
			log.Println("Parity: Even")
		}
	} else {
		log.Printf("COM port is not listen!\n")
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
	default:
		serialStopBits = serial.OneStopBit
	}
	return
}
