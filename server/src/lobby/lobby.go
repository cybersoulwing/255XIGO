package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

type Character struct {
	ID    uint32
	Name  string
	Job   string
	Level uint8
}

type LobbySession struct {
	Conn           net.Conn
	SelectedCharID uint32
}

var characters = []Character{
	{ID: 1, Name: "Alice", Job: "WAR", Level: 50},
	{ID: 2, Name: "Bob", Job: "BLM", Level: 45},
}

func sendCharList(s *LobbySession) {
	// パケット例: [キャラ数][ID,NameLen,Name,JobLen,Job,Level]...
	buf := make([]byte, 1024)
	buf[0] = byte(len(characters))
	offset := 1
	for _, c := range characters {
		binary.LittleEndian.PutUint32(buf[offset:], c.ID)
		offset += 4
		buf[offset] = byte(len(c.Name))
		offset++
		copy(buf[offset:], c.Name)
		offset += len(c.Name)
		buf[offset] = byte(len(c.Job))
		offset++
		copy(buf[offset:], c.Job)
		offset += len(c.Job)
		buf[offset] = c.Level
		offset++
	}
	s.Conn.Write(buf[:offset])
	fmt.Println("CharList sent")
}

func receiveCharSelect(s *LobbySession) {
	buf := make([]byte, 4)
	n, _ := s.Conn.Read(buf)
	if n >= 4 {
		s.SelectedCharID = binary.LittleEndian.Uint32(buf)
		fmt.Println("Selected CharID:", s.SelectedCharID)
	}
}

func sendMapServerInfo(s *LobbySession) {
	// MAPサーバIP/Portを返す (例: 127.0.0.1:54230)
	buf := make([]byte, 6)
	buf[0] = 127
	buf[1] = 0
	buf[2] = 0
	buf[3] = 1
	binary.LittleEndian.PutUint16(buf[4:], 54230)
	s.Conn.Write(buf)
	fmt.Println("MapServerInfo sent")
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	session := &LobbySession{Conn: conn}
	sendCharList(session)
	receiveCharSelect(session)
	sendMapServerInfo(session)
}

func main() {
	ln, err := net.Listen("tcp", ":55000") // Lobbyサーバ固定ポート
	if err != nil {
		panic(err)
	}
	fmt.Println("Lobby server listening on :55000")
	for {
		conn, _ := ln.Accept()
		go handleConnection(conn)
	}
}
