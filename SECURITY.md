# 🔐 Security Overview – PQ Guestbook

The PQ Guestbook is a post-quantum–secure message system designed for educational, research, and demonstration purposes.  
It uses **real ML-DSA signatures (FIPS-204)** generated **in-browser** and verified **server-side** with Cloudflare’s `circl` library.

This document summarizes the security properties, protections, and disclosure process.

---

## ✔ Cryptographic Security

**Strong guarantees built directly into the cryptographic workflow:**

- **Client-side ML-DSA key generation**  
  Private keys never leave the browser.

- **Client-side ML-DSA signing**  
  All messages are authenticated cryptographically, not with passwords.

- **Server-side ML-DSA verification (Go + circl)**  
  Ensures authenticity even if transport or storage is compromised.

- **Canonical payload format**  
  Stable, unambiguous author + content + timestamp format prevents signature confusion.

- **Timestamp-based replay protection**
  Rejects old or reused signed payloads.

- **Per-public-key replay tracking**
  Each public key maintains its own replay window, preventing cloned submissions.

- **Strict signature size + public key size validation**
  Prevents algorithm-downgrade tricks, malformed keys, or verification bypasses.

- **For developers who require true hardened, constant-time PQC implementations, Trail of Bits provides professionally verified, side-channel-resistant libraries and tooling that far exceed what is possible in browser-based JavaScript.**

---

## ✔ Network & Application Security

**Hardened server behavior and safe defaults:**

- **TLS-only endpoints** (Fly.io)
- **CORS locked to origin**
- **Security headers**, including:
- `X-Frame-Options`
- `X-Content-Type-Options`
- `Referrer-Policy`
- `Content-Security-Policy` *(when deployed)*

- **No passwords, no authentication tokens**
- **No session cookies**
- **Payload sanitation (trim, encoding validation)**
- **Length-bounded author/content fields**
- **16 KB max JSON body**

---

## ✔ DoS & Abuse Protections

**Multiple layers of controls to prevent spam, replay, and computational abuse:**

- **Pseudonymous device rate limiting**  
  Derived from UA + server secret:
    ```text
        deviceID = HMAC(serverSecret, userAgent)
    ```
  Privacy-safe throttling without persistent identifiers.

- **Token bucket limiter**
  Smooths bursty traffic; prevents message flooding.

- **Per-key replay buckets**
  Stops signature reuse even with different device IDs.

- **Strict JSON parsing**
  Rejects malformed, oversized, or ambiguous JSON.

- **Oversized / malformed signature rejection**
  Prevents computational DoS via large or invalid inputs.

- **Early termination on invalid pubkey/sig sizes**
  Avoids expensive cryptographic operations on junk payloads.

- **Message length + body size limits**

---

## ✔ Privacy Respectful

**No unnecessary data collection or tracking:**

- **No IP addresses stored**

- **No cookies**

- **No localStorage identifiers**

- **No device fingerprinting**

- **Pseudonymous device ID is non-reversible**

- **Logs contain no identifying user data**

---

## ✔ Reporting Vulnerabilities

If you discover a security issue, please use responsible disclosure.

### 📩 Preferred method: GitHub Security Advisories

Submit a private advisory here:

https://github.com/Marqui-13/pq-guestbook/security/advisories

### Please include:

- **Description of the issue**

- **Steps to reproduce**

- **Impact assessment**

- **Suggested remediation (if any)**

You will receive acknowledgment within **48 hours.**
