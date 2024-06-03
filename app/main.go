package main

import (
	"flag"
	"fmt"
	"net"
)

func main() {
	var resolver string
	flag.StringVar(&resolver, "resolver", "1.1.1.1:53", "DNS resolver to forward queries to")
	flag.Parse()

	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		fmt.Println("Failed to resol[]byteve UDP address:", err)
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
		questions, _ := ParseQuestions(buf[:size], requestHeader.QDCOUNT)

		answers := ForwardQuestions(resolver, questions, requestHeader.ID)

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
			QDCOUNT: 0,
			ANCOUNT: 0,
			NSCOUNT: requestHeader.NSCOUNT,
			ARCOUNT: requestHeader.ARCOUNT,
		}

		response := MakeMessage(header)

		for _, question := range questions {
			response.AddQuestion(question)
		}

		for _, answer := range answers {
			response.AddAnswer(answer)
		}

		_, err = udpConn.WriteToUDP(response.Bytes(), source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
