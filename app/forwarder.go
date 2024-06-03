package main

import (
	"fmt"
	"net"
)

func ForwardQuestions(addr string, questions []DNSQuestion, id uint16) []DNSAnswer {
	answers := []DNSAnswer{}

	udpConn, err := net.Dial("udp", addr)
	if err != nil {
		fmt.Println("Failed to connect to address:", err)
		return answers
	}

	defer udpConn.Close()

	for _, question := range questions {
		questionBytes := question.Bytes()

		header := DNSHeader{
			ID:      id,
			QR:      0,
			OPCODE:  0,
			AA:      0,
			TC:      0,
			RD:      1,
			RA:      0,
			Z:       0,
			RCODE:   0,
			QDCOUNT: 1,
			ANCOUNT: 0,
			NSCOUNT: 0,
			ARCOUNT: 0,
		}

		headerBytes := header.Bytes()
		buf := append(headerBytes, questionBytes...)

		_, err = udpConn.Write(buf)
		if err != nil {
			fmt.Println("Failed to send data:", err)
			return answers
		}

		responseBuf := make([]byte, 512)

		n, err := udpConn.Read(responseBuf)
		if err != nil {
			fmt.Println("Failed to receive data:", err)
			return answers
		}

		responseHeader := ParseHeader(responseBuf[:n])
		answers = append(answers, ParseAnswers(responseBuf[:n], responseHeader.QDCOUNT, responseHeader.ANCOUNT)...)
	}

	return answers
}
