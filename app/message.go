package main

import (
	"bytes"
	"encoding/binary"
	"strings"
)

type DNSHeader struct {
	ID      uint16
	QR      uint8
	OPCODE  uint8
	AA      uint8
	TC      uint8
	RD      uint8
	RA      uint8
	Z       uint8
	RCODE   uint8
	QDCOUNT uint16
	ANCOUNT uint16
	NSCOUNT uint16
	ARCOUNT uint16
}

type DNSMessage struct {
	Header    DNSHeader
	Questions []DNSQuestion
	Answers   []DNSAnswer
}

type DNSAnswer struct {
	Name     []byte
	Type     uint16
	Class    uint16
	TTL      uint32
	RDLength uint16
	RData    []byte
}

type DNSQuestion struct {
	Name  []byte
	Type  uint16
	Class uint16
}

func (m *DNSHeader) Bytes() []byte {
	bytes := make([]byte, 12)
	binary.BigEndian.PutUint16(bytes[0:], m.ID)
	binary.BigEndian.PutUint16(bytes[2:4], m.combineFlags())
	binary.BigEndian.PutUint16(bytes[4:6], m.QDCOUNT)
	binary.BigEndian.PutUint16(bytes[6:8], m.ANCOUNT)
	binary.BigEndian.PutUint16(bytes[8:10], m.NSCOUNT)
	binary.BigEndian.PutUint16(bytes[10:12], m.ARCOUNT)
	return bytes
}

func (m *DNSHeader) combineFlags() uint16 {
	return uint16(m.QR)<<15 | uint16(m.OPCODE)<<11 | uint16(m.AA)<<10 |
		uint16(m.TC)<<9 | uint16(m.RD)<<8 | uint16(m.RA)<<7 | uint16(m.Z)<<4 |
		uint16(m.RCODE)
}

func (m *DNSMessage) AddQuestion(q DNSQuestion) {
	m.Questions = append(m.Questions, q)
	m.Header.QDCOUNT = m.Header.QDCOUNT + 1
}

func (m *DNSMessage) Bytes() []byte {
	headerBytes := m.Header.Bytes()
	bytes := []byte{}
	bytes = append(bytes, headerBytes...)

	for _, question := range m.Questions {
		questionBytes := question.Bytes()
		bytes = append(bytes, questionBytes...)
	}

	for _, answer := range m.Answers {
		answerBytes := answer.Bytes()
		bytes = append(bytes, answerBytes...)
	}
	return bytes
}

func (a *DNSAnswer) Bytes() []byte {
	bytes := []byte{}
	bytes = append(bytes, a.Name...)
	bytes = binary.BigEndian.AppendUint16(bytes, a.Type)
	bytes = binary.BigEndian.AppendUint16(bytes, a.Class)
	bytes = binary.BigEndian.AppendUint32(bytes, a.TTL)
	bytes = binary.BigEndian.AppendUint16(bytes, a.RDLength)
	bytes = binary.BigEndian.AppendUint32(bytes, binary.BigEndian.Uint32(a.RData))
	return bytes
}

func (m *DNSMessage) AddAnswer(a DNSAnswer) {
	m.Answers = append(m.Answers, a)
	m.Header.ANCOUNT = m.Header.ANCOUNT + 1
}

func (q *DNSQuestion) Bytes() []byte {
	bytes := []byte{}
	bytes = append(bytes, q.Name...)
	bytes = binary.BigEndian.AppendUint16(bytes, q.Type)
	bytes = binary.BigEndian.AppendUint16(bytes, q.Class)
	return bytes
}

func MakeMessage(header DNSHeader) DNSMessage {
	return DNSMessage{Header: header, Questions: []DNSQuestion{}, Answers: []DNSAnswer{}}
}

func MakeAnswer(name []byte, rdata []byte) DNSAnswer {
	return DNSAnswer{Name: name, Type: 1, Class: 1, TTL: 60, RDLength: uint16(len(rdata)), RData: rdata}
}

func ParseHeader(buf []byte) DNSHeader {
	return DNSHeader{
		ID:      binary.BigEndian.Uint16(buf[0:2]),
		QR:      uint8(buf[2] >> 7),
		OPCODE:  uint8(buf[2] >> 3 & 0x0f),
		AA:      uint8(buf[2] >> 2 & 0x01),
		TC:      uint8(buf[2] >> 1 & 0x01),
		RD:      uint8(buf[2] & 0x01),
		RA:      uint8(buf[3] >> 7),
		Z:       uint8(buf[3] >> 4 & 0x07),
		RCODE:   uint8(buf[3] & 0x0f),
		QDCOUNT: binary.BigEndian.Uint16(buf[4:6]),
		ANCOUNT: binary.BigEndian.Uint16(buf[6:8]),
		NSCOUNT: binary.BigEndian.Uint16(buf[8:10]),
		ARCOUNT: binary.BigEndian.Uint16(buf[10:12]),
	}
}

func MakeQuestion(name []byte) DNSQuestion {
	return DNSQuestion{Name: name, Type: 1, Class: 1}
}

func ParseQuestions(buf []byte, questionCount uint16) ([]DNSQuestion, int) {
	questions := []DNSQuestion{}
	offset := 12

	for i := 0; i < int(questionCount); i++ {
		len := bytes.Index(buf[offset:], []byte{0})
		label := ParseDomain(buf[offset:offset+len+1], buf)
		question := MakeQuestion(label)
		questions = append(questions, question)
		offset += len + 1 + 4
	}

	return questions, offset
}

func ParseDomain(data []byte, source []byte) []byte {
	domain := decodeDNSPacket(data, source)
	segments := strings.Split(domain, ".")
	var encodedDomain []byte

	for _, segment := range segments {
		encodedDomain = append(encodedDomain, byte(len(segment)))
		encodedDomain = append(encodedDomain, []byte(segment)...)
	}

	encodedDomain = append(encodedDomain, 0x00)
	return encodedDomain
}

func decodeDNSPacket(packet []byte, source []byte) string {
	offset := 0
	labels := []string{}

	for {
		if packet[offset] == 0 {
			break
		}

		if (packet[offset]&0xc0)>>6 == 0b11 {
			pointer := int(binary.BigEndian.Uint16(packet[offset:offset+2]) << 2 >> 2)
			length := bytes.Index(source[pointer:], []byte{0})
			labels = append(labels, decodeDNSPacket(source[pointer:pointer+length+1], source))
			offset += 2
			continue
		}

		length := int(packet[offset])
		substring := packet[offset+1 : offset+1+length]
		labels = append(labels, string(substring))
		offset += length + 1
	}

	return strings.Join(labels, ".")
}

func ParseAnswers(buf []byte, questionCount uint16, answerCount uint16) []DNSAnswer {
	answers := []DNSAnswer{}
	_, offset := ParseQuestions(buf, questionCount)

	for i := 0; i < int(answerCount); i++ {
		len := bytes.Index(buf[offset:], []byte{0})
		label := ParseDomain(buf[offset:offset+len+1], buf)
		offset += len + 1

		answer := DNSAnswer{
			Name: label,
		}

		answer.Type = extractUint16(buf, &offset)
		answer.Class = extractUint16(buf, &offset)
		answer.TTL = extractUint32(buf, &offset)
		answer.RDLength = extractUint16(buf, &offset)
		answer.RData = extractBytes(buf, &offset, int(answer.RDLength))
		answers = append(answers, answer)
	}

	return answers
}

func extractBytes(src []byte, offset *int, length int) []byte {
	result := src[*offset : *offset+length]
	*offset += length
	return result
}

func extractUint16(src []byte, offset *int) uint16 {
	result := []byte{src[*offset], src[*offset+1]}
	*offset += 2
	return binary.BigEndian.Uint16(result[:])
}

func extractUint32(src []byte, offset *int) uint32 {
	result := [4]byte{src[*offset], src[*offset+1], src[*offset+2], src[*offset+3]}
	*offset += 4
	return binary.BigEndian.Uint32(result[:])
}
