package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"time"
)

type Player struct {
	ID       uint32
	X, Y, Z  float32
	Rotation uint8
}

type Session struct {
	Conn   net.Conn
	Player *Player
}

// ZoneInパケット
func SendZoneIn(s *Session) {
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint16(buf[0:2], 0x00D) // ZoneIn opcode
	binary.LittleEndian.PutUint16(buf[2:4], uint16(len(buf)))
	s.Conn.Write(buf)
	fmt.Println("ZoneIn sent")
}

// Spawnパケット
func SendSpawn(s *Session) {
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint16(buf[0:2], 0x00F) // Spawn opcode
	binary.LittleEndian.PutUint16(buf[2:4], uint16(len(buf)))
	binary.LittleEndian.PutUint32(buf[4:8], s.Player.ID)
	s.Conn.Write(buf)
	fmt.Println("Spawn sent")
}

// CharUpdateパケット
func SendCharUpdate(s *Session) {
	p := s.Player
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint16(buf[0:2], 0x00A) // CharUpdate opcode
	binary.LittleEndian.PutUint16(buf[2:4], uint16(len(buf)))

	binary.LittleEndian.PutUint32(buf[4:8], p.ID)
	binary.LittleEndian.PutUint32(buf[8:12], math.Float32bits(p.X))
	binary.LittleEndian.PutUint32(buf[12:16], math.Float32bits(p.Y))
	binary.LittleEndian.PutUint32(buf[16:20], math.Float32bits(p.Z))
	buf[20] = p.Rotation

	s.Conn.Write(buf)
	fmt.Println("CharUpdate sent")
}

// ログイン後処理
func OnLogin(s *Session) {
	SendZoneIn(s)
	time.Sleep(50 * time.Millisecond)

	SendSpawn(s)
	time.Sleep(50 * time.Millisecond)

	SendCharUpdate(s)
}

// 接続ハンドラ
func handleConnection(conn net.Conn) {
	defer conn.Close()

	player := &Player{
		ID:       1234,
		X:        100.0,
		Y:        0.0,
		Z:        100.0,
		Rotation: 0,
	}

	session := &Session{
		Conn:   conn,
		Player: player,
	}

	fmt.Println("Player connected:", player.ID)
	OnLogin(session)
}

// サーバ起動（ポート54230固定）
func main() {
	ln, err := net.Listen("tcp", ":54230")
	if err != nil {
		panic(err)
	}
	fmt.Println("MAP server listening on :54230")

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}
		go handleConnection(conn)
	}
}
