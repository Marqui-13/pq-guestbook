package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	mldsa44 "github.com/cloudflare/circl/sign/mldsa/mldsa44"
	mldsa65 "github.com/cloudflare/circl/sign/mldsa/mldsa65"
	mldsa87 "github.com/cloudflare/circl/sign/mldsa/mldsa87"
)

type Message struct {
	Author    string `json:"author"`
	Content   string `json:"content"`
	Timestamp int64  `json:"ts"`
	Algo      string `json:"algo"`
	Signature string `json:"sig"`
	PubKey    string `json:"pubkey"`
	Browser   string `json:"browser"`  // e.g. "Chrome 120"
	Platform  string `json:"platform"` // e.g. "Windows 10"
	UserAgent string `json:"ua"`       // optional full UA
}

var (
	messages []Message
	mu       sync.RWMutex
)

func main() {
	// Serve static frontend
	http.Handle("/", http.FileServer(http.Dir("static")))

	// API: get all messages
	http.HandleFunc("/api/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "GET only", 405)
			return
		}
		mu.RLock()
		defer mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	})

	// API: post new message with pure ML-DSA signature
	http.HandleFunc("/api/post", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST only", 405)
			return
		}
		var m Message
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(w, "bad json", 400)
			return
		}

		// Decode client-sent pubkey and sig
		pubBytes, err := base64.RawStdEncoding.DecodeString(m.PubKey)
		if err != nil {
			http.Error(w, "invalid pubkey encoding", 400)
			return
		}
		sigBytes, err := base64.RawStdEncoding.DecodeString(m.Signature)
		if err != nil {
			http.Error(w, "invalid signature encoding", 400)
			return
		}

		// Reconstruct signed bytes
		toVerify := []byte(m.Author + m.Content + fmt.Sprintf("%d", m.Timestamp))

		// Select ML-DSA level based on pubkey size and verify
		var valid bool
		switch len(pubBytes) {
		case mldsa44.PublicKeySize:
			var pub mldsa44.PublicKey
			if err := pub.UnmarshalBinary(pubBytes); err != nil {
				http.Error(w, "invalid pubkey", 400)
				return
			}
			valid = mldsa44.Verify(&pub, toVerify, nil, sigBytes)
		case mldsa65.PublicKeySize:
			var pub mldsa65.PublicKey
			if err := pub.UnmarshalBinary(pubBytes); err != nil {
				http.Error(w, "invalid pubkey", 400)
				return
			}
			valid = mldsa65.Verify(&pub, toVerify, nil, sigBytes)
		case mldsa87.PublicKeySize:
			var pub mldsa87.PublicKey
			if err := pub.UnmarshalBinary(pubBytes); err != nil {
				http.Error(w, "invalid pubkey", 400)
				return
			}
			valid = mldsa87.Verify(&pub, toVerify, nil, sigBytes)
		default:
			http.Error(w, "unsupported ML-DSA level", 400)
			return
		}

		if !valid {
			http.Error(w, "invalid ML-DSA signature", 401)
			return
		}

		mu.Lock()
		messages = append([]Message{m}, messages...) // Newest first
		mu.Unlock()

		w.WriteHeader(200)
		w.Write([]byte(`{"status":"quantum-safe post accepted"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("⚛️ Post-Quantum Guestbook live on :" + port + " (real ML-DSA browser signing)")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
