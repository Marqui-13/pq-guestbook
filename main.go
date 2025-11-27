package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

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
	replayMu sync.Mutex
	seen     = make(map[string]map[int64]bool)
	rateMu     sync.Mutex
	rateLimits = make(map[string]*rateInfo)
)

type rateInfo struct {
	Tokens     float64
	LastRefill time.Time
}

// Device-ID HMAC, modify if testing locally
var rateLimitSecret = []byte(os.Getenv("RATE_LIMIT_SECRET"))

func canonicalPayload(author, content string, ts int64) []byte {
	// Prevents injection or mismatched signing order
	return []byte(fmt.Sprintf("%s\n%s\n%d", author, content, ts))
}

// Returns true if this pubkey+timestamp combo has been seen before
func replaySeen(pubkey []byte, ts int64) bool {
	replayMu.Lock()
	defer replayMu.Unlock()

	key := hex.EncodeToString(pubkey)
	if _, ok := seen[key]; !ok {
		seen[key] = make(map[int64]bool)
	}

	if seen[key][ts] {
		return true
	}
	seen[key][ts] = true
	return false
}

func deviceIDFromUA(ua string) string {
	mac := hmac.New(sha256.New, rateLimitSecret)
	mac.Write([]byte(ua))
	sum := mac.Sum(nil)
	return hex.EncodeToString(sum[:8]) // 16-char stable pseudonym
}

const (
	maxTokens  = 8    // Burst
	refillRate = 0.25 // tokens/sec (1 request every 4s sustained)
)

func allowDevice(id string) bool {
	rateMu.Lock()
	defer rateMu.Unlock()

	now := time.Now()
	ri, ok := rateLimits[id]
	if !ok {
		rateLimits[id] = &rateInfo{
			Tokens:     maxTokens,
			LastRefill: now,
		}
		return true
	}

	elapsed := now.Sub(ri.LastRefill).Seconds()
	ri.LastRefill = now

	ri.Tokens = math.Min(maxTokens, ri.Tokens+elapsed*refillRate)
	if ri.Tokens < 1 {
		return false
	}

	ri.Tokens -= 1
	return true
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "https://pq-guestbook.fly.dev")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, User-Agent")

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
}

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

	const (
		maxBodyBytes  = 16 * 1024 // 16 KB total JSON body
		maxAuthorLen  = 80
		maxContentLen = 2000
	)

	// API: post new message with pure ML-DSA signature
	http.HandleFunc("/api/post", func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w)

		if r.Method == http.MethodOptions {
			w.WriteHeader(200)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", 405)
			return
		}

		// Hard cap body size
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

		// Parse JSON
		var m Message
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(w, "bad json", 400)
			return
		}

		// Sanitize input
		m.Author = strings.TrimSpace(m.Author)
		m.Content = strings.TrimSpace(m.Content)

		if m.Author == "" || m.Content == "" {
			http.Error(w, "author/content required", 400)
			return
		}
		if len(m.Author) > maxAuthorLen || len(m.Content) > maxContentLen {
			http.Error(w, "message too long", 413)
			return
		}

		// Decode keys and signature
		pubBytes, err := base64.RawStdEncoding.DecodeString(m.PubKey)
		if err != nil || len(pubBytes) > 5000 {
			http.Error(w, "invalid pubkey", 400)
			return
		}
		sigBytes, err := base64.RawStdEncoding.DecodeString(m.Signature)
		if err != nil || len(sigBytes) > 5000 {
			http.Error(w, "invalid signature", 400)
			return
		}

		// Timestamp replay prevention (±15s)
		now := time.Now().UnixMilli()
		drift := now - m.Timestamp
		if drift < -15000 || drift > 15000 {
			http.Error(w, "timestamp not fresh", 401)
			return
		}

		// Per-pubkey replay detection
		if replaySeen(pubBytes, m.Timestamp) {
			http.Error(w, "replay detected", 401)
			return
		}

		// Device rate limit (UA fingerprint)
		deviceID := deviceIDFromUA(m.UserAgent)
		if !allowDevice(deviceID) {
			http.Error(w, "rate limit exceeded", 429)
			return
		}

		// Canonical message format
		canonical := canonicalPayload(m.Author, m.Content, m.Timestamp)

		// ML-DSA verify
		var valid bool
		switch len(pubBytes) {
		case mldsa44.PublicKeySize:
			var pub mldsa44.PublicKey
			if pub.UnmarshalBinary(pubBytes) != nil {
				http.Error(w, "invalid pubkey", 400)
				return
			}
			valid = mldsa44.Verify(&pub, canonical, nil, sigBytes)

		case mldsa65.PublicKeySize:
			var pub mldsa65.PublicKey
			if pub.UnmarshalBinary(pubBytes) != nil {
				http.Error(w, "invalid pubkey", 400)
				return
			}
			valid = mldsa65.Verify(&pub, canonical, nil, sigBytes)

		case mldsa87.PublicKeySize:
			var pub mldsa87.PublicKey
			if pub.UnmarshalBinary(pubBytes) != nil {
				http.Error(w, "invalid pubkey", 400)
				return
			}
			valid = mldsa87.Verify(&pub, canonical, nil, sigBytes)

		default:
			http.Error(w, "unsupported ML-DSA level", 400)
			return
		}

		if !valid {
			http.Error(w, "invalid ML-DSA signature", 401)
			return
		}

		// Store messages
		mu.Lock()
		messages = append([]Message{m}, messages...)
		mu.Unlock()

		w.WriteHeader(200)
		w.Write([]byte(`{"status":"quantum-safe post accepted"}`))
	})

	// Ensure RATE_LIMIT_SECRET is set
	secret := os.Getenv("RATE_LIMIT_SECRET")
	if secret == "" {
		log.Fatal("RATE_LIMIT_SECRET is not set. Set it using `fly secrets set RATE_LIMIT_SECRET=$(openssl rand -hex 32)`")
	}

	// Decode hex → bytes (Fly secrets store raw strings)
	secretBytes, err := hex.DecodeString(secret)
	if err != nil {
		log.Fatalf("RATE_LIMIT_SECRET must be a 32-byte hex string: %v", err)
	}

	if len(secretBytes) != 32 {
		log.Fatalf("RATE_LIMIT_SECRET must decode to exactly 32 bytes, got %d bytes", len(secretBytes))
	}

	rateLimitSecret = secretBytes

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("⚛️ Post-Quantum Guestbook live on :" + port + " (real ML-DSA browser signing)")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
