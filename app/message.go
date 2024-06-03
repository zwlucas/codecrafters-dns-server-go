package main

import "encoding/binary"

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
	Header   DNSHeader
	Question []byte
	Answer   DNSAnswer
}

type DNSAnswer struct {
	Name     []byte
	Type     uint16
	Class    uint16
	TTL      uint32
	RDLength uint16
	RData    []byte
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

func (m *DNSMessage) AddQuestion(q []byte) {
	m.Question = q
	m.Question = binary.BigEndian.AppendUint16(m.Question, 1)
	m.Question = binary.BigEndian.AppendUint16(m.Question, 1)
}

func (m *DNSMessage) Bytes() []byte {
	headerBytes := m.Header.Bytes()
	answerBytes := m.Answer.Bytes()
	bytes := []byte{}
	bytes = append(bytes, headerBytes...)
	bytes = append(bytes, m.Question...)
	bytes = append(bytes, answerBytes...)
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

func MakeMessage(header DNSHeader) DNSMessage {
	return DNSMessage{Header: header, Question: []byte{}, Answer: DNSAnswer{}}
}

func (m *DNSMessage) AddAnswer(a DNSAnswer) {
	m.Answer = a
	m.Header.ANCOUNT = 1
}

func MakeAnswer(name []byte, rdata []byte) DNSAnswer {
	return DNSAnswer{Name: name, Type: 1, Class: 1, TTL: 60, RDLength: uint16(len(rdata)), RData: rdata}
}
