package main

import (
	"fmt"
	"net"
)

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		fmt.Println("Failed to resolve UDP address:", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return
	}
	defer udpConn.Close()

	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		receivedData := string(buf[:size])
		fmt.Printf("Received %d bytes from %s: %s\n", size, source, receivedData)
		requestHeader := ParseHeader(buf[:size])

		rcode := uint8(4)
		if requestHeader.OPCODE == 0 {
			rcode = 0
		}

		header := DNSHeader{
			ID:      requestHeader.ID,
			QR:      1,
			OPCODE:  requestHeader.OPCODE,
			AA:      0,
			TC:      0,
			RD:      requestHeader.RD,
			RA:      0,
			Z:       0,
			RCODE:   rcode,
			QDCOUNT: 1,
			ANCOUNT: 0,
			NSCOUNT: 0,
			ARCOUNT: 0,
		}

		response := MakeMessage(header)
		response.AddQuestion([]byte("\x0ccodecrafters\x02io\x00"))

		answer := MakeAnswer([]byte("\x0ccodecrafters\x02io\x00"), []byte("\x08\x08\x08\x08"))
		response.AddAnswer(answer)

		_, err = udpConn.WriteToUDP(response.Bytes(), source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
