package main

import (
	"crypto/rand"
	"testing"
	"crypto"
	mldsa44 "github.com/cloudflare/circl/sign/mldsa/mldsa44"
	mldsa65 "github.com/cloudflare/circl/sign/mldsa/mldsa65"
	mldsa87 "github.com/cloudflare/circl/sign/mldsa/mldsa87"
)

// ------------------------- ML-DSA-44 -------------------------

func BenchmarkMLDSA44(b *testing.B) {
	msg := []byte("benchmark message payload")

	// Keygen benchmark
	b.Run("KeyGen", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, _ = mldsa44.GenerateKey(nil)
		}
	})

	// Sign benchmark
	pk, sk, _ := mldsa44.GenerateKey(nil)
	b.Run("Sign", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = sk.Sign(rand.Reader, msg, crypto.Hash(0))
		}
	})

	// Verify benchmark
	sig, _ := sk.Sign(rand.Reader, msg, crypto.Hash(0))
	b.Run("Verify", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = mldsa44.Verify(pk, msg, nil, sig)
		}
	})
}

// ------------------------- ML-DSA-65 -------------------------

func BenchmarkMLDSA65(b *testing.B) {
	msg := []byte("benchmark message payload")

	b.Run("KeyGen", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, _ = mldsa65.GenerateKey(nil)
		}
	})

	pk, sk, _ := mldsa65.GenerateKey(nil)
	b.Run("Sign", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = sk.Sign(rand.Reader, msg, crypto.Hash(0))
		}
	})

	sig, _ := sk.Sign(rand.Reader, msg, crypto.Hash(0))
	b.Run("Verify", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = mldsa65.Verify(pk, msg, nil, sig)
		}
	})
}

// ------------------------- ML-DSA-87 -------------------------

func BenchmarkMLDSA87(b *testing.B) {
	msg := []byte("benchmark message payload")

	b.Run("KeyGen", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, _ = mldsa87.GenerateKey(nil)
		}
	})

	pk, sk, _ := mldsa87.GenerateKey(nil)
	b.Run("Sign", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = sk.Sign(rand.Reader, msg, crypto.Hash(0))
		}
	})

	sig, _ := sk.Sign(rand.Reader, msg, crypto.Hash(0))
	b.Run("Verify", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = mldsa87.Verify(pk, msg, nil, sig)
		}
	})
}
