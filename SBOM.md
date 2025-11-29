# **📦 SBOM.md — Software Bill of Materials for pq-guestbook**

This document describes the Software Bill of Materials (SBOM) for the **Post-Quantum Guestbook** project, how it is generated, and how to interpret the results.

The SBOM provides transparency into the components, dependencies, and build artifacts used in both the **Go backend** and the **static browser-based frontend**.
This project emphasizes **minimalism, security, and auditability**, so its SBOM reflects that.

Full-project SBOM file is located in this project's root directory: **sbom-full-project.cdx.json**

---

## **1. What an SBOM Represents**

An SBOM (Software Bill of Materials) lists:

* **Packages / modules used**
* **Versions**
* **Licenses**
* **Files included in the build**
* **Build tools**
* **Cryptographic libraries and dependencies**

It is required for software supply-chain security and is becoming mandatory in many security compliance frameworks (NIST, US EO 14028, CISA SSDF, PCI-DSS 4.0, etc.).

This project uses **CycloneDX 1.6**, a widely adopted security-focused SBOM specification.

---

# **2. Tools Used**

### **Syft** (Anchore)

Installed via Chocolatey (Windows):

```sh
choco install syft
```

Syft is used because it:

* Works with Go binaries and modules
* Supports CycloneDX output
* Supports container + directory scanning
* Produces deterministic output (good for reproducibility)

---

# **3. Project Structure & SBOM Implications**

Project layout:

```
pq-guestbook/
│  main.go
│  go.mod
│  go.sum
│  Dockerfile
│  fly.toml
│  SECURITY.md
│  LICENSE.md
│  README.md
│  SBOM.md
│  CBOM.md
│  mldsa_bench_test.go
│  pqgb.exe (built binary)
│
└──static/
       index.html
       favicon.ico
└──images/
       og-image.jpeg
       dashboard1.PNG
       dashboard2.PNG
       dashboard3.PNG
       dashboard4.PNG
       dashboard5.PNG
       frontendbm1.PNG
       frontendbm2.PNG
       frontendbm3.PNG
       mldsa_benchmarks.PNG

```

### **Backend:**

* Pure Go module
* Uses **Cloudflare CIRCL** for ML-DSA
* No CGO
* No external system libraries
* SBOM reflects real dependencies (`go.mod` + compiled binary)

### **Frontend:**

* Pure static files (HTML + JS + CSS)
* **No npm**
* **No node_modules**
* **No package.json**
* Imports noble-post-quantum from CDN at runtime:

```js
import { ml_dsa65 } from "https://esm.sh/@noble/post-quantum";
```

Because these files are **not local**, Syft correctly produces a **minimal SBOM**.

---

# **4. Backend SBOM Generation**

From project root:

### **A. Go module SBOM**

```sh
syft dir:. --output cyclonedx-json > sbom-backend.json
```

### **B. Compiled binary SBOM**

```sh
syft pqgb.exe --output cyclonedx-json > sbom-binary.json
```

Cloudflare’s CIRCL library appears in this SBOM under Go modules.

---

# **5. Frontend SBOM Generation**

From `static/`:

```sh
syft dir:. --output cyclonedx-json > sbom-frontend.json
```

### **Why it looks nearly empty**

Because the frontend is:

* Pure static content
* Contains **no locally installed JS dependencies**
* Has **no package manager**
* Has **no build artifacts**
* Pulls cryptographic code from a remote CDN (esm.sh)

Syft **does not** fetch or analyze CDN-hosted packages, so the SBOM contains only metadata about the local directory.

---

# **6. Combined SBOM (Full Project)**

Recommended:

```sh
syft dir:. --output cyclonedx-json > sbom-full.json
```

This will include:

* Go backend modules
* Static directory metadata
* Tool metadata

But still **no external JS dependencies**, by design.

---

# **7. How To Interpret the Results**

### **Minimal frontend SBOM is expected**

This project intentionally uses **zero build tools** and loads crypto libraries at runtime, so the SBOM reflects:

* No JavaScript dependencies
* No supply-chain entries
* No versioned packages

This is desirable for a small, privacy-preserving, CDN-driven app.

### **Backend SBOM is authoritative**

CIRCL modules appear and define the cryptographic supply chain.

---

# **8. Optional Enhancements**

### **A. Add manual CycloneDX “external component” entries**

For CDN packages:

```json
{
  "type": "library",
  "name": "@noble/post-quantum",
  "version": "latest",
  "purl": "pkg:npm/%40noble/post-quantum"
}
```

### **B. Switch to an npm-based build**

This enables:

* Full dependency graph
* Version pinning
* Reproducibility
* Rich SBOMs

### **C. Add SPDX license manifest**

Syft can generate SPDX tags for frontend assets as well.

---

# **9. Security Benefits of Having SBOMs**

* Enables vulnerability scanning
* Supports supply-chain audits
* Necessary for enterprise PQC deployments
* Aids PQC researchers auditing cryptographic dependencies

---

# **10. Summary**

* **Backend SBOM** contains Go + CIRCL cryptographic dependencies
* **Frontend SBOM** is intentionally minimal (because no local JS deps)
* **Full-project SBOM** combines both
* **CDN crypto library is not included by design**

This results in an SBOM that is clean, minimal, secure, and highly auditable which exactly aligns with the project’s PQC onboarding purpose.
