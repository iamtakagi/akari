package main

import (
	"encoding/binary"
	"log"
	"net"
	"time"

	"golang.org/x/net/ipv4"
)

var ntpEpoch = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)

type NtpPacket struct {
	Settings           uint8
	Stratum            uint8
	Poll               uint8
	Precision          uint8
	RootDelay          uint32
	RootDispersion     uint32
	ReferenceID        uint32
	ReferenceTimestamp [8]byte
	OriginateTimestamp [8]byte
	ReceiveTimestamp   [8]byte
	TransmitTimestamp  [8]byte
}

func toNtpTime(t time.Time) (ntpTime [8]byte) {
	seconds := uint32(t.Sub(ntpEpoch).Seconds())
	fraction := uint32((t.Sub(ntpEpoch) % time.Second) * (1 << 32 / time.Second))
	binary.BigEndian.PutUint32(ntpTime[:4], seconds)
	binary.BigEndian.PutUint32(ntpTime[4:], fraction)
	return ntpTime
}

func NewNtpPacket(receivedPacket NtpPacket, now time.Time) *NtpPacket {
	return &NtpPacket{
		Settings:           0x24,
		Stratum:            1,
		Poll:               receivedPacket.Poll,
		Precision:          receivedPacket.Precision,
		RootDelay:          0,
		RootDispersion:     0,
		ReferenceID:        0,
		ReferenceTimestamp: toNtpTime(now),
		OriginateTimestamp: receivedPacket.TransmitTimestamp,
		ReceiveTimestamp:   toNtpTime(now),
		TransmitTimestamp:  toNtpTime(now),
	}
}

func handleRequest(conn *net.UDPConn, p *ipv4.PacketConn) {
	b := make([]byte, 48)
	for {
		_, _, remoteAddr, err := p.ReadFrom(b)
		if err != nil {
			log.Fatalf("読み取りに失敗: %v", err)
		}

		var ntpPacket *NtpPacket

		receivedPacket := NtpPacket{}
		receivedPacket.Settings = b[0]
		receivedPacket.Stratum = b[1]
		receivedPacket.Poll = b[2]
		receivedPacket.Precision = b[3]
		receivedPacket.RootDelay = binary.BigEndian.Uint32(b[4:8])
		receivedPacket.RootDispersion = binary.BigEndian.Uint32(b[8:12])
		receivedPacket.ReferenceID = binary.BigEndian.Uint32(b[12:16])
		copy(receivedPacket.ReferenceTimestamp[:], b[16:24])
		copy(receivedPacket.OriginateTimestamp[:], b[24:32])
		copy(receivedPacket.ReceiveTimestamp[:], b[32:40])
		copy(receivedPacket.TransmitTimestamp[:], b[40:48])

		mode := b[0] & 0x7

		now := time.Now()

		switch mode {

		case 3:
			log.Println("受信 <- " + remoteAddr.String())
			log.Printf("%+v", receivedPacket)
			ntpPacket = NewNtpPacket(receivedPacket, now)
			copy(b[32:40], ntpPacket.TransmitTimestamp[:])
		default:
			log.Printf("非対応のモードです: %v", mode)
		}

		b[0] = ntpPacket.Settings
		b[1] = ntpPacket.Stratum
		b[2] = ntpPacket.Poll
		b[3] = ntpPacket.Precision
		binary.BigEndian.PutUint32(b[4:8], ntpPacket.RootDelay)
		binary.BigEndian.PutUint32(b[8:12], ntpPacket.RootDispersion)
		binary.BigEndian.PutUint32(b[12:16], ntpPacket.ReferenceID)
		copy(b[16:24], ntpPacket.ReferenceTimestamp[:])
		copy(b[24:32], ntpPacket.OriginateTimestamp[:])
		copy(b[32:40], ntpPacket.ReceiveTimestamp[:])
		copy(b[40:48], ntpPacket.TransmitTimestamp[:])

		log.Println("送信 -> " + remoteAddr.String())

		log.Printf("%+v", ntpPacket)

		_, err = conn.WriteTo(b, remoteAddr)
		if err != nil {
			log.Fatalf("書き込みに失敗: %v", err)
		}
	}
}

func main() {
	addr, err := net.ResolveUDPAddr("udp", ":123")
	if err != nil {
		log.Fatalf("アドレス解決に失敗: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("起動に失敗: %v", err)
	}
	defer conn.Close()

	p := ipv4.NewPacketConn(conn)
	defer p.Close()

	handleRequest(conn, p)
}
