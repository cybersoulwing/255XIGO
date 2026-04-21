package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"os"
	"time"
)

// クライアントから送られるログインリクエスト
type LoginRequest struct {
	Username          string        `json:"username"`                      // ユーザー名
	Password          string        `json:"password"`                      // パスワード
	OTP               string        `json:"otp"`                           // ワンタイムパスワード
	NewPassword       string        `json:"new_password"`                  // 新しいパスワード
	Version           []interface{} `json:"version"`                       // バージョン情報（配列）
	Command           int8          `json:"command"`                       // コマンド種別
	TrustToken        string        `json:"trust_token,omitempty"`         // 信頼トークン
	TrustThisComputer bool          `json:"trust_this_computer,omitempty"` // このPCを信頼するか
}

// サーバーが返すログイン応答
type LoginReply struct {
	Result       int8     `json:"result"`                  // 結果コード
	ErrorMessage string   `json:"error_message,omitempty"` // エラーメッセージ（省略可）
	AccountID    uint32   `json:"account_id,omitempty"`    // アカウントID
	SessionHash  [16]byte `json:"session_hash,omitempty"`  // セッションハッシュ
	TOTPUri      string   `json:"TOTP_uri,omitempty"`      // TOTP URI
}

func main() {
	certFile := "server.crt" // TLS証明書ファイル
	keyFile := "server.key"  // 秘密鍵ファイル

	// 証明書が存在しなければ自動生成
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		log.Println("TLS証明書が無いため自動生成します...")
		generateSelfSignedCert(certFile, keyFile)
	}

	// 証明書をロード
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("TLS証明書読み込み失敗: %v", err)
	}

	// TLS設定
	config := &tls.Config{Certificates: []tls.Certificate{cert}}

	// TCP + TLSでリッスン
	ln, err := tls.Listen("tcp", "127.0.0.1:54231", config)
	if err != nil {
		log.Fatalf("TLSリッスン失敗: %v", err)
	}
	defer ln.Close()
	log.Println("Login server listening on 127.0.0.1:54231")

	// 接続受付ループ
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("接続受け入れエラー:", err)
			continue
		}
		go handleConnection(conn) // ゴルーチンで並行処理
	}
}

// 個別接続処理
func handleConnection(conn net.Conn) {
	defer conn.Close()
	log.Printf("New connection from %s", conn.RemoteAddr())

	// データ読み取り
	buf := make([]byte, 8192)
	n, err := conn.Read(buf)
	if err != nil {
		log.Println("読み取りエラー:", err)
		return
	}

	// JSONをデコード
	var loginReq LoginRequest
	if err := json.Unmarshal(buf[:n], &loginReq); err != nil {
		log.Println("JSON decode エラー:", err)
		return
	}

	log.Printf("Login request: %+v\n", loginReq)

	// デフォルト返信（エラー）
	reply := LoginReply{
		Result:       0x02,
		ErrorMessage: "",
	}

	// コマンドごとの応答
	switch loginReq.Command {
	case 0x10: // login
		reply.Result = 0x01
		reply.AccountID = 12345
		rand.Read(reply.SessionHash[:])
	case 0x20: // create account
		reply.Result = 0x03
	case 0x30: // change password
		reply.Result = 0x06
	case 0x31: // create TOTP
		reply.Result = 0x10
		reply.TOTPUri = "otpauth://totp/xiloader:testuser?secret=JBSWY3DPEHPK3PXP&issuer=xiloader"
	case 0x32: // remove TOTP
		reply.Result = 0x12
	}

	// JSONエンコード
	data, err := json.Marshal(reply)
	if err != nil {
		log.Println("JSON encode エラー:", err)
		return
	}

	data = append(data, '\n') // 改行追加
	_, err = conn.Write(data) // クライアントへ送信
	if err != nil {
		log.Println("送信エラー:", err)
		return
	}

	log.Printf("Login JSON reply sent to %s", conn.RemoteAddr())
}

// 自己署名TLS証明書生成
func generateSelfSignedCert(certFile, keyFile string) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("秘密鍵生成失敗: %v", err)
	}

	// 証明書テンプレート
	serialNumber, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"GoLoginServer"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// 証明書作成
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("証明書生成失敗: %v", err)
	}

	// PEMで保存
	certOut, _ := os.Create(certFile)
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, _ := os.Create(keyFile)
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()
}
