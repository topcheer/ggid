# Confidential Computing (Intel SGX / AMD SEV) for IAM Systems

> Research document for the GGID project — protecting keys, PII, and authentication
> logic using Trusted Execution Environments (TEEs), with a focus on Intel SGX and
> AMD SEV-SNP. Covers enclave-based key storage, remote attestation, Go SDK
> landscape, performance benchmarks, and a concrete migration roadmap from the
> current PEM-file key model toward hardware-attested confidential computing.
>
> **Companion documents:** `hsm-kms-integration.md` covers HSM/KMS integration
> and the `CryptoProvider` interface design. `key-rotation-iam.md` covers the
> `RotatingKeyProvider` lifecycle. `secret-management-iam.md` covers Vault and
> environment-variable secret patterns. This document focuses on **confidential
> computing** — protecting data *in use* inside CPU-encrypted memory.

**Status:** Draft
**Audience:** GGID architects, security researchers, platform engineers, compliance officers
**Last Updated:** 2025

---

## Table of Contents

1. [Confidential Computing Concepts](#1-confidential-computing-concepts)
2. [Intel SGX Deep Dive](#2-intel-sgx-deep-dive)
3. [AMD SEV-SNP](#3-amd-sev-snp)
4. [Enclave-Based Key Storage](#4-enclave-based-key-storage)
5. [Attestation Flow](#5-attestation-flow)
6. [Go SGX/TEE SDK Landscape](#6-go-sgxtee-sdk-landscape)
7. [Performance Overhead Analysis](#7-performance-overhead-analysis)
8. [When to Use Confidential Computing vs HSM](#8-when-to-use-confidential-computing-vs-hsm)
9. [Confidential Computing for Multi-Tenant IAM](#9-confidential-computing-for-multi-tenant-iam)
10. [Remote Attestation Integration](#10-remote-attestation-integration)
11. [GGID Confidential Computing Roadmap](#11-ggid-confidential-computing-roadmap)
12. [Gap Analysis & Recommendations](#12-gap-analysis--recommendations)
13. [References](#13-references)

---

## 1. Confidential Computing Concepts

### 1.1 The Three States of Data

Data exists in three states, each requiring different protection mechanisms:

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Data Protection Across States                     │
├──────────────┬─────────────────┬──────────────────┬────────────────┤
│    State     │   Description   │ Protection Layer │  IAM Examples   │
├──────────────┼─────────────────┼──────────────────┼────────────────┤
│ Data at Rest │ Stored on disk  │ Disk encryption, │ PostgreSQL TDE  │
│              │                 │ TDE, backup encr │ (pgcrypto)      │
├──────────────┼─────────────────┼──────────────────┼────────────────┤
│ Data in      │ Network transit │ TLS 1.3, mTLS    │ gRPC TLS, HTTPS │
│ Transit      │ between systems │                  │ Gateway→Service │
├──────────────┼─────────────────┼──────────────────┼────────────────┤
│ Data in Use  │ Active in RAM,  │ CONFIDENTIAL     │ JWT signing key │
│              │ CPU registers   │ COMPUTING (TEE)  │ PII in memory   │
│              │ during compute  │                  │ Password hash   │
└──────────────┴─────────────────┴──────────────────┴────────────────┘
```

Traditional security covers data at rest (encryption) and data in transit (TLS).
**Confidential computing** addresses the third state: data in use. When data
resides in RAM or CPU registers during computation, it is in plaintext and
visible to the operating system, hypervisor, cloud administrator, and anyone
who can inspect memory.

### 1.2 Trusted Execution Environment (TEE)

A Trusted Execution Environment is a secure area of a main processor that
guarantees:

1. **Confidentiality** — Memory inside the TEE is encrypted by the CPU. The
   hypervisor, OS kernel, and even a physical attacker with a bus snooper
   cannot read the contents.

2. **Integrity** — Any modification of TEE memory by non-TEE code is detected
   and causes a hardware fault. The TEE can detect tampering.

3. **Attestation** — The TEE can cryptographically prove to a remote party
   that it is running a specific piece of code on genuine hardware. This
   creates a hardware root of trust.

```
┌───────────────────────────────────────────────────┐
│                    CPU Package                     │
│  ┌─────────────────────────────────────────────┐   │
│  │           Memory Encryption Engine          │   │
│  │     (AES-XTS in hardware, per-page keys)     │   │
│  └──────────────────┬──────────────────────────┘   │
│                     │                               │
│  ┌──────────────────▼──────────────────────────┐   │
│  │              Normal Memory                   │   │
│  │  ┌──────────────────────────────────────┐    │   │
│  │  │        OS / Hypervisor / VMM         │    │   │
│  │  │  ┌──────────────────────────────┐    │    │   │
│  │  │  │     Untrusted Application     │    │    │   │
│  │  │  └──────────────────────────────┘    │    │   │
│  │  └──────────────────────────────────────┘    │   │
│  │                                              │   │
│  │  ┌──────────────────────────────────────┐    │   │
│  │  │      PROTECTED MEMORY (Encrypted)     │    │   │
│  │  │                                      │    │   │
│  │  │   ┌──────────────────────────────┐   │    │   │
│  │  │   │        TEE / Enclave         │   │    │   │
│  │  │   │                              │   │    │   │
│  │  │   │  • Private keys              │   │    │   │
│  │  │   │  • PII (email, phone)        │   │    │   │
│  │  │   │  • Password hash computation │   │    │   │
│  │  │   │  • Token signing logic       │   │    │   │
│  │  │   │                              │   │    │   │
│  │  │   │  Keys NEVER leave plaintext  │   │    │   │
│  │  │   └──────────────────────────────┘   │    │   │
│  │  └──────────────────────────────────────┘    │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

### 1.3 Enclave

An **enclave** is the specific instantiation of a TEE for a particular
application. In Intel SGX, an enclave is a segment of protected memory
into which code and data are loaded. The enclave's memory is encrypted
and access-controlled at the hardware level.

Key enclave properties:
- **Entry points are controlled** — Only specific functions designated
  as enclave entry points (`ECALL`s in SGX terminology) can be called
  from outside.
- **Code measurement** — The enclave's initial state (code + data) is
  cryptographically hashed during creation. This measurement (`MRENCLAVE`
  in SGX) is part of the attestation report.
- **Sealing** — Data can be encrypted ("sealed") using a key derived
  from the enclave's measurement, allowing it to be persisted and later
  decrypted only by the same enclave (or a designated successor).
- **No I/O** — Enclaves cannot directly perform I/O (file, network,
  console). They must use `OCALL`s to ask the untrusted host to perform
  I/O on their behalf.

### 1.4 Why Confidential Computing Matters for IAM

IAM systems are uniquely valuable targets because they concentrate:

1. **Root keys** — JWT signing keys, OAuth client secrets, SAML
   signing certificates. Compromise of a JWT signing key enables
   forging tokens for any user in the system.

2. **PII at scale** — Email addresses, phone numbers, full names,
   organizational memberships. A memory dump of an IAM service
   reveals every user's identity.

3. **Credential material** — Password hashes (even with Argon2id, the
   hash itself is sensitive), TOTP seeds, WebAuthn challenge data.

4. **Session tokens** — Active refresh tokens, authorization codes,
   session cookies — all present in service memory.

**Threat model without confidential computing:**

```
┌─────────────────────────────────────────────────────────────────┐
│                     Attack Surfaces on IAM Keys                  │
│                                                                  │
│  Cloud Admin ──► Can read VM memory ──► Sees JWT signing key    │
│       │                                                          │
│       ├─► Can read disk ──► Sees PEM private key file           │
│       │                                                          │
│       └─► Can snapshot VM ──► Full memory dump                  │
│                                                                  │
│  Compromised OS ──► ptrace/process_vm_readv ──► Key in memory   │
│       │                                                          │
│       ├─► /proc/<pid>/mem ──► Read process memory directly      │
│       │                                                          │
│       └─► Kernel module ──► Read any process memory             │
│                                                                  │
│  Hypervisor 0-day ──► VM escape ──► Guest memory access         │
│                                                                  │
│  Cold boot attack ──► RAM contents persist ──► Key recovery    │
└─────────────────────────────────────────────────────────────────┘
```

With confidential computing, the signing key is generated *inside* the
enclave, used *inside* the enclave, and never appears in plaintext in
normal memory. Even a cloud administrator with full VM access cannot
extract the key.

### 1.5 TEE vs HSM: Conceptual Comparison

| Property | HSM (FIPS 140-2 L3) | TEE (SGX / SEV-SNP) |
|---|---|---|
| **What it protects** | Cryptographic keys | Entire computation (code + data) |
| **Physical tamper resistance** | Active tamper detection, zeroization | None (relies on CPU security) |
| **FIPS certification** | Yes (140-2, 140-3) | No (NIST review in progress) |
| **Key storage** | Inside HSM, non-exportable | Inside enclave, sealed to enclave |
| **Application logic protection** | No — app runs in normal memory | Yes — entire app runs in enclave |
| **PII protection** | No — PII in app memory | Yes — PII in encrypted memory |
| **Throughput** | ~10,000 RSA-2048 ops/sec | ~1,000+ RSA-2048 ops/sec in enclave |
| **Deployment complexity** | PKCS#11 integration, network HSM | Recompile for SGX, or use CVM |
| **Cost** | $5,000–$50,000 per HSM | Cloud surcharge (~10–25%) |
| **Attestation** | Physical chain of custody | Cryptographic attestation report |
| **Best for** | Key storage, CA operations | Full application protection |

**Key insight:** HSMs protect *keys*. TEEs protect *computation*. For
an IAM system, you need both: HSM for the root CA and TEE for the
authentication service that processes passwords and issues tokens.

---

## 2. Intel SGX Deep Dive

### 2.1 Architecture

Intel Software Guard Extensions (SGX) is a set of x86-64 instruction
extensions that allow applications to create enclaves — protected
memory regions whose contents are encrypted and integrity-protected
by the CPU.

```
┌──────────────────────────────────────────────────────────┐
│                  Application Process                      │
│                                                          │
│  ┌─────────────────────────┐                            │
│  │    Untrusted Code        │                            │
│  │   (normal memory)        │                            │
│  │                         │                            │
│  │  • HTTP server           │                            │
│  │  • Database driver       │                            │
│  │  • Business logic        │                            │
│  └────────┬───────┬────────┘                            │
│           │ ECALL │ OCALL                                │
│  ┌────────▼───────▼────────┐                            │
│  │     Trusted Code         │                            │
│  │   (EPC — Enclave Page    │                            │
│  │    Cache, encrypted)     │                            │
│  │                         │                            │
│  │  • Key generation        │     MRENCLAVE: hash of     │
│  │  • JWT signing           │     initial enclave code   │
│  │  • Password hashing      │     and data               │
│  │  • Private key storage   │                            │
│  │                         │     MRSIGNER: hash of the   │
│  │  All memory encrypted    │     enclave signing key    │
│  │  by Memory Encryption    │                            │
│  │  Engine (MEE)            │                            │
│  └─────────────────────────┘                            │
│                                                          │
│  EPC boundary: ~128 MB (SGX1), ~512 MB (SGX1 with EPC    │
│  swelling), up to 1 TB (SGX2 with oversubscription)      │
└──────────────────────────────────────────────────────────┘
```

### 2.2 Enclave Page Cache (EPC)

The EPC is the encrypted portion of DRAM reserved for enclave memory.
It is managed by the CPU's Memory Encryption Engine (MEE):

- **SGX1**: Fixed EPC size, typically 128 MB (usable ~96 MB after
  metadata). This is a hard limit — if an enclave needs more than
  the EPC, it must swap pages out and re-encrypt them, which is
  extremely slow.

- **SGX2 (Ice Lake / Tiger Lake / Sapphire Rapids)**: Dynamic
  memory management via `EAUG` (Extend/Add Pages), `EMODPR`
  (Modify Page Permissions), and `EREMOVE` instructions. EPC can
  grow up to 1 TB with oversubscription support.

For IAM workloads, EPC sizing matters:
- RSA-2048 key: ~1 KB
- JWKS response: ~2 KB
- Argon2id hash (64 MB working set): **Does NOT fit in SGX1 EPC**
- 1000 active session tokens: ~500 KB
- User PII cache (10,000 users × 500 bytes): ~5 MB

**Critical limitation:** Argon2id with 64 MB memory cost (as configured
in GGID's `pkg/crypto/crypto.go`) cannot run inside an SGX1 enclave.
The 64 MB working set exceeds the ~96 MB usable EPC when combined with
enclave code and data. This forces either:
1. Using SGX2 hardware with larger EPC
2. Reducing Argon2id memory cost when running inside the enclave
3. Running password hashing outside the enclave (loses protection)
4. Using a different KDF inside the enclave

### 2.3 SGX1 vs SGX2

| Feature | SGX1 | SGX2 |
|---|---|---|
| **EPC size** | 128 MB typical | Up to 1 TB |
| **Dynamic pages** | No — fixed at creation | Yes (`EAUG`, `EREMOVE`) |
| **Page permissions** | Fixed | Runtime modification (`EMODPR`) |
| **Oversubscription** | Manual, very slow | Hardware-assisted |
| **CPU support** | Skylake, Kaby Lake, Coffee Lake | Ice Lake SP+, Tiger Lake+, Sapphire Rapids |
| **Cloud availability** | Azure DCsv2 (limited) | Azure DCsv3/DCsv4, AWS (6th gen), GCP |
| **Thread management** | Fixed at creation | Dynamic thread add/remove |

### 2.4 Attestation

Attestation is the process by which an enclave proves its identity and
state to a remote party. SGX supports two models:

#### 2.4.1 Local Attestation

Used when two enclaves on the same platform need to prove their
identities to each other:

```
Enclave A                    Enclave B
    │                            │
    │  1. EREPORT (B's REPORT)   │
    │ ─────────────────────────► │
    │                            │
    │  2. REPORT (contains MAC   │
    │     key derived from CPU)  │
    │ ◄───────────────────────── │
    │                            │
    │  3. EGETKEY (derive key)   │
    │     Verify REPORT MAC      │
    │                            │
    │  Verified: B's MRENCLAVE,  │
    │  MRSIGNER, attributes      │
```

The `EREPORT` instruction generates a hardware-signed `REPORT` structure.
The `EGETKEY` instruction derives a MAC key from the CPU's internal
keys that both enclaves can compute.

#### 2.4.2 Remote Attestation (EPID)

EPID (Enhanced Privacy ID) is an Intel-managed group signature scheme.
The enclave generates a "quote" — a signed attestation report — that
can be verified using Intel's EPID verification infrastructure.

```
Enclave           QE (Quoting Enclave)       Intel IAS (Attestation Service)
   │                    │                            │
   │ 1. EREPORT(QE)     │                            │
   │ ─────────────────► │                            │
   │                    │                            │
   │                    │ 2. Generate Quote          │
   │                    │    (EPID signature)        │
   │ 3. Quote ◄───────── │                            │
   │                    │                            │
   │                    │ 4. Send Quote to IAS       │
   │                    │ ──────────────────────────► │
   │                    │                            │
   │                    │     5. IAS verifies EPID   │
   │                    │        signature, returns  │
   │                    │        attestation report  │
   │                    │ ◄────────────────────────── │
   │ 6. Report ◄────────│                            │
```

EPID is **deprecated** for new deployments. Intel recommends DCAP.

#### 2.4.3 Remote Attestation (DCAP)

Data Center Attestation Primitives (DCAP) replaces the Intel-managed
EPID with a local verification model. The cloud provider or data center
runs its own Provisioning Certificate Caching Service (PCCS):

```
┌─────────────┐    ┌──────────────┐    ┌──────────────────┐
│   Enclave   │    │   QE (SGX)   │    │  DCAP PCCS       │
│             │    │              │    │  (cloud provider) │
│  1.EREPORT  │───►│              │    │                  │
│             │    │ 2.Quote gen  │    │ 3.Cached Intel   │
│             │    │   + sig      │    │   cert chain     │
│             │    │              │◄──►│  (no Intel IAS   │
│             │    │              │    │   round trip)    │
└──────┬──────┘    └──────┬───────┘    └──────────────────┘
       │                  │
       │                  │
       ▼                  ▼
┌──────────────────────────────────┐
│         Remote Verifier           │
│  (your application or service)    │
│                                  │
│  4. Verify quote signature using  │
│     Intel root CA → PCK cert →    │
│     QE cert → Quote               │
│                                  │
│  5. Check MRENCLAVE matches       │
│     expected enclave measurement  │
│                                  │
│  6. Extract enclave-held secrets  │
│     via secure channel            │
└──────────────────────────────────┘
```

DCAP advantages:
- No round-trip to Intel's servers
- Cloud provider caches certificates
- Lower latency for attestation
- Supports offline verification

### 2.5 Security Properties

SGX provides:

1. **Memory confidentiality** — EPC pages are encrypted with keys
   derived from the CPU. Even physical DRAM inspection reveals
   ciphertext only.

2. **Memory integrity** — Each EPC page has a MAC (Message
   Authentication Code) stored in a hardware Merkle tree. Any
   modification by non-enclave code is detected.

3. **Enclave identity** — `MRENCLAVE` is a SHA-256 hash of the
   enclave's code and initial data. It changes if even one byte of
   code is modified.

4. **Non-oracle property** — The enclave cannot be forced to execute
   arbitrary code; only designated entry points can be called.

### 2.6 Known Attacks

SGX has been subject to extensive academic security research. Several
side-channel and microarchitectural attacks have been published:

| Attack | Year | Class | Impact | Mitigation |
|---|---|---|---|---|
| **Foreshadow (L1TF)** | 2018 | Speculative execution | Read enclave memory via L1 cache | Microcode update + SGX disable on vulnerable CPUs |
| **Foreshadow-NG** | 2018 | Speculative execution | Read enclave memory from other enclaves | Same as Foreshadow |
| **LVI (Load Value Injection)** | 2020 | Speculative execution | Inject values into enclave computation | Microcode + compiler mitigations |
| **SGAxe** | 2020 | Cache timing | Extract attestation keys from QE | Patched in DCAP |
| **Plundervolt** | 2020 | Voltage manipulation | Fault injection breaks enclave crypto | BIOS voltage lock |
| **CacheOut** | 2020 | Cache side-channel | Cross-enclave data leak | Microcode update |
| **CrossLine** | 2021 | Enclave boundary bypass | Breaks enclave isolation on SGX1 | Use SGX2 |
| **AEPIC Leak** | 2022 | APIC MMIO | Read stale EPC data from APIC | Microcode update |
| **SQUIP** | 2023 | SMT contention | Infer sibling thread enclave activity | Disable SMT or use SGX2 |

**Key takeaway:** SGX security is a cat-and-mouse game. Intel patches
vulnerabilities via microcode updates, but some require disabling SGX
entirely on affected CPUs. IAM systems using SGX must:
- Run on latest-generation hardware (Sapphire Rapids or newer)
- Apply all microcode updates
- Consider SMT (HyperThreading) disabling for high-assurance enclaves
- Monitor CVEs for SGX-related vulnerabilities

---

## 3. AMD SEV-SNP

### 3.1 Architecture

AMD Secure Encrypted Virtualization - Secure Nested Paging (SEV-SNP)
takes a fundamentally different approach from SGX. Instead of protecting
individual application enclaves, SEV-SNP encrypts entire virtual machines:

```
┌─────────────────────────────────────────────────────────────┐
│                  Physical Host (AMD EPYC)                    │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              AMD Secure Processor (PSP)                │  │
│  │  • Manages encryption keys for each VM                │  │
│  │  • Generates attestation reports                      │  │
│  │  • Firmware-level root of trust                       │  │
│  └───────────────────────┬───────────────────────────────┘  │
│                          │                                   │
│  ┌───────────────────────▼───────────────────────────────┐  │
│  │          Memory Encryption Engine (AES-XTS)           │  │
│  │     Each VM gets a unique key, transparent to VM      │  │
│  └───────────────────────┬───────────────────────────────┘  │
│                          │                                   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                    Hypervisor / VMM                   │   │
│  │  • Sees encrypted memory for all VMs                  │   │
│  │  • Cannot read guest memory even with host access     │   │
│  │  • Page table control is hardware-validated (RMP)     │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  Guest VM 1   │  │  Guest VM 2   │  │  Guest VM 3   │      │
│  │  (Encrypted)  │  │  (Encrypted)  │  │  (Encrypted)  │      │
│  │               │  │               │  │               │      │
│  │  GGID Auth    │  │  GGID Policy  │  │  GGID OAuth   │      │
│  │  Full kernel   │  │  Full kernel   │  │  Full kernel   │      │
│  │  Full app stack│  │  Full app stack│  │  Full app stack│      │
│  │               │  │               │  │               │      │
│  │  Key: K1       │  │  Key: K2       │  │  Key: K3       │      │
│  │  (from PSP)    │  │  (from PSP)    │  │  (from PSP)    │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 SEV-SNP vs SEV vs SEV-ES

AMD's confidential computing has evolved through three generations:

| Feature | SEV (Naples) | SEV-ES (Rome) | SEV-SNP (Milan+) |
|---|---|---|---|
| **Memory encryption** | Yes | Yes | Yes |
| **Encrypted state** (CPU registers on exit) | No | Yes | Yes |
| **Integrity protection** (RMP) | No | No | **Yes** |
| **Hypervisor attack resistance** | Weak | Moderate | **Strong** |
| **Attestation** | Platform report | Platform report | **VM-level report** |
| **CPU support** | EPYC 7001 | EPYC 7002/7003 | EPYC 7003+ |

SEV-SNP is the first generation that provides both confidentiality *and*
integrity, making it suitable for security-critical workloads.

### 3.3 Reverse-Map Table (RMP)

The RMP is SEV-SNP's integrity mechanism. It is a hardware-managed table
that maps each physical page to the VM that owns it:

```
┌──────────────────────────────────────────────────────────┐
│                  RMP Entry (per physical page)            │
│                                                          │
│  ┌──────────────┬──────────────┬──────────┬────────────┐ │
│  │  ASID (VM)   │  GPA (Guest  │ Assigned │  Page Size  │ │
│  │              │  Physical    │ (valid?) │            │ │
│  │              │  Address)    │          │            │ │
│  └──────────────┴──────────────┴──────────┴────────────┘ │
│                                                          │
│  Hardware checks RMP on every memory access:             │
│                                                          │
│  Hypervisor writes to page:                              │
│    1. CPU looks up RMP for physical page                 │
│    2. If ASID != hypervisor ASID → FAULT (denied)        │
│    3. If page is assigned to a guest → RMP_NPF guest     │
│                                                          │
│  This prevents:                                          │
│    • Hypervisor remapping guest pages                    │
│    • Hypervisor writing to guest memory                  │
│    • DMA attacks on guest memory                         │
│    • Page table manipulation by hypervisor               │
└──────────────────────────────────────────────────────────┘
```

### 3.4 Hardware-Validated Page Table

In SEV-SNP, the guest VM's page table entries are validated by the
hardware. The hypervisor cannot change page mappings to redirect
guest memory accesses. This is a critical integrity guarantee that
earlier SEV versions lacked.

### 3.5 Why SEV-SNP is Easier to Deploy than SGX

```
┌──────────────────────────────────────────────────────────────┐
│                 Deployment Complexity                         │
│                                                               │
│  SGX:                                                         │
│  ┌──────────────────────────────────────────────────────┐    │
│  │  1. Write C/C++ enclave code                          │    │
│  │  2. Define ECALL/OCALL interface                      │    │
│  │  3. Compile enclave with SGX SDK (special toolchain)  │    │
│  │  4. Sign enclave (MRSIGNER)                           │    │
│  │  5. Port untrusted code to use ECALLs                 │    │
│  │  6. Handle EPC size limits (swap, pagination)         │    │
│  │  7. Implement attestation verification                │    │
│  │  8. Test side-channel resistance                      │    │
│  │  → MAJOR REFACTOR REQUIRED                           │    │
│  └──────────────────────────────────────────────────────┘    │
│                                                               │
│  SEV-SNP:                                                     │
│  ┌──────────────────────────────────────────────────────┐    │
│  │  1. Deploy VM on SEV-SNP-capable cloud instance      │    │
│  │  2. Enable SEV-SNP in VM launch parameters           │    │
│  │  3. (Optional) Implement attestation verification     │    │
│  │  → NO CODE CHANGES REQUIRED                          │    │
│  └──────────────────────────────────────────────────────┘    │
│                                                               │
│  Confidential Containers (Kubernetes):                       │
│  ┌──────────────────────────────────────────────────────┐    │
│  │  1. Annotate pod: "io.katacontainers.confidential"   │    │
│  │  2. Cloud provider schedules on TEE-capable node     │    │
│  │  3. Attestation handled by runtime class             │    │
│  │  → MINIMAL MANIFEST CHANGES                          │    │
│  └──────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────┘
```

For GGID, SEV-SNP (or Azure Confidential VMs / AWS Nitro Enclaves)
offers a pragmatic path: run the entire microservice suite inside
confidential VMs with zero code changes, gaining encrypted memory
protection for all PII, keys, and tokens.

### 3.6 SEV-SNP Attestation

SEV-SNP attestation provides a VM-level attestation report:

```
┌───────────────────────────────────────────────────┐
│              SEV-SNP Attestation Flow              │
│                                                    │
│  Guest VM            AMD PSP        Verifier       │
│     │                   │               │          │
│     │ 1. SNP_GET_REPORT │               │          │
│     │ ─────────────────►│               │          │
│     │                   │               │          │
│     │ 2. Report (signed │               │          │
│     │    by PSP with    │               │          │
│     │    VCEK key)      │               │          │
│     │ ◄─────────────────│               │          │
│     │                   │               │          │
│     │ 3. Report + VCEK certificate      │          │
│     │    (from AMD KDS)                 │          │
│     │ ─────────────────────────────────►│          │
│     │                   │               │          │
│     │                   │  4. Verify:   │          │
│     │                   │  - VCEK chain │          │
│     │                   │  - Report sig │          │
│     │                   │  - MEASUREMENT│          │
│     │                   │  - FAMILY_ID  │          │
│     │                   │  - IMAGE_ID   │          │
│     │                   │               │          │
│     │                   │  5. Issue     │          │
│     │                   │   attested    │          │
│     │                   │   token/secret│          │
│     │ ◄────────────────────────────────│          │
└───────────────────────────────────────────────────┘
```

The report contains:
- **MEASUREMENT** — Hash of the VM's initial firmware state
- **FAMILY_ID / IMAGE_ID** — Developer-defined identifiers
- **REPORT_DATA** — 64 bytes of caller-provided data (nonce/challenge)
- **POLICY** — Security policy flags (SMT, migration, etc.)
- **VCEK signature** — Signed by the Versioned Chip Endorsement Key

---

## 4. Enclave-Based Key Storage

### 4.1 Current GGID Key Storage

GGID currently stores keys in two ways:

1. **RSA signing key** — PEM file on disk (`configs/rsa_private.pem`),
   loaded at startup by `loadOrCreateKeyProvider()` in both Auth and
   OAuth services. The key is held in process memory as an
   `*rsa.PrivateKey` struct.

2. **Password pepper** — Set from an environment variable via
   `SetPepper()`. Stored as a global `[]byte` variable.

3. **Audit hash chain secret** — Set from config via
   `SetHashChainSecret()`. Stored as a global `[]byte` variable.

4. **RotatingKeyProvider** — `RotateKey()` generates new RSA keys in
   memory using `crypto/rand`. The key never touches disk — but it
   is in plaintext process memory.

**Vulnerability:** Any of these secrets can be extracted by anyone with
process memory access: cloud admin, compromised OS, hypervisor exploit.

### 4.2 Enclave-Based Key Storage Pattern

In an enclave model, the key lifecycle changes fundamentally:

```
┌──────────────────────────────────────────────────────────────┐
│              Enclave-Based Key Lifecycle                      │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐     │
│  │  Phase 1: Key Generation (inside enclave)           │     │
│  │                                                     │     │
│  │  crypto/rand → RSA key → stored in EPC memory       │     │
│  │  Key NEVER leaves enclave in plaintext              │     │
│  │  No disk file, no env var, no process memory        │     │
│  └─────────────────────────┬───────────────────────────┘     │
│                            │                                  │
│  ┌─────────────────────────▼───────────────────────────┐     │
│  │  Phase 2: Key Sealing (for persistence)             │     │
│  │                                                     │     │
│  │  Enclave derives sealing key from MRENCLAVE         │     │
│  │  Encrypts private key with sealing key              │     │
│  │  Writes sealed blob to disk via OCALL               │     │
│  │  Only same-enclave (same MRENCLAVE) can decrypt     │     │
│  └─────────────────────────┬───────────────────────────┘     │
│                            │                                  │
│  ┌─────────────────────────▼───────────────────────────┐     │
│  │  Phase 3: Key Use (inside enclave)                  │     │
│  │                                                     │     │
│  │  JWT signing:                                       │     │
│  │    1. Untrusted code passes JWT claims to enclave    │     │
│  │    2. Enclave marshals JWT, signs with private key   │     │
│  │    3. Enclave returns signed token to untrusted code │     │
│  │    4. Private key never in untrusted memory          │     │
│  └─────────────────────────┬───────────────────────────┘     │
│                            │                                  │
│  ┌─────────────────────────▼───────────────────────────┐     │
│  │  Phase 4: Key Rotation (inside enclave)             │     │
│  │                                                     │     │
│  │  1. Generate new key in enclave                     │     │
│  │  2. Seal new key, overwrite sealed blob             │     │
│  │  3. Demote old key to "previous" (still in EPC)     │     │
│  │  4. After grace period, zeroize old key             │     │
│  └─────────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────────┘
```

### 4.3 Go Code: Enclave Key Manager Pattern

The following code shows how an enclave-based key manager would integrate
with GGID's existing `KeyProvider` interface:

```go
// Package enclavekey provides a KeyProvider implementation that delegates
// key operations to a TEE (Trusted Execution Environment) enclave.
//
// This is a design sketch — actual SGX integration requires compiling
// the enclave code with the Intel SGX SDK and using EGo or Gramine for
// Go support. This interface shows how the existing domain.KeyProvider
// pattern extends to enclave-based key storage.
package enclavekey

import (
	"crypto/rsa"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
)

// EnclaveSigner defines the interface for TEE-based signing operations.
// In a real implementation, this would make ECALLs to the SGX enclave.
type EnclaveSigner interface {
	// GenerateKey generates a new RSA key inside the enclave.
	// The key is never exposed to the host process.
	GenerateKey(bits int) (keyID string, err error)

	// Sign signs data with the enclave-held private key.
	// The private key never leaves the enclave.
	Sign(keyID string, data []byte) ([]byte, error)

	// PublicKey retrieves the public key from the enclave.
	// Only the public key is returned — the private key stays inside.
	PublicKey(keyID string) (*rsa.PublicKey, error)

	// SealKey encrypts the private key with a key derived from the
	// enclave's MRENCLAVE measurement. The sealed blob can be persisted
	// to disk and only decrypted by the same enclave.
	SealKey(keyID string) ([]byte, error)

	// UnsealKey loads a sealed key blob back into the enclave.
	UnsealKey(sealed []byte) (keyID string, err error)

	// RotateKey generates a new key and seals it for persistence.
	RotateKey() (newKeyID string, err error)

	// Attest generates an attestation report proving the enclave's
	// identity to a remote party.
	Attest(reportData [64]byte) ([]byte, error)

	// DestroyKey zeroizes a key inside the enclave.
	DestroyKey(keyID string) error
}

// EnclaveKeyProvider implements domain.KeyProvider by delegating
// all signing operations to an SGX/SEV enclave. The private key
// is never available to the host Go process.
type EnclaveKeyProvider struct {
	mu          sync.RWMutex
	enclave     EnclaveSigner
	currentID   string
	previousID  string
	rotatedAt   time.Time
	gracePeriod time.Duration
}

// Compile-time interface check.
var _ domain.KeyProvider = (*EnclaveKeyProvider)(nil)

// NewEnclaveKeyProvider creates a key provider backed by a TEE enclave.
// It generates the initial key inside the enclave and seals it for
// persistence across restarts.
func NewEnclaveKeyProvider(enclave EnclaveSigner, sealedKey []byte, gracePeriod time.Duration) (*EnclaveKeyProvider, error) {
	if gracePeriod == 0 {
		gracePeriod = 24 * time.Hour
	}

	kp := &EnclaveKeyProvider{
		enclave:     enclave,
		gracePeriod: gracePeriod,
	}

	var keyID string
	var err error

	if len(sealedKey) > 0 {
		// Restore existing key from sealed blob.
		keyID, err = enclave.UnsealKey(sealedKey)
		if err != nil {
			slog.Warn("failed to unseal enclave key, generating new", "error", err)
			keyID, err = enclave.GenerateKey(2048)
			if err != nil {
				return nil, fmt.Errorf("enclave key generation: %w", err)
			}
		}
		slog.Info("enclave key restored from sealed blob", "kid", keyID)
	} else {
		// Generate new key inside enclave.
		keyID, err = enclave.GenerateKey(2048)
		if err != nil {
			return nil, fmt.Errorf("enclave key generation: %w", err)
		}
		slog.Info("enclave key generated inside TEE", "kid", keyID)
	}

	kp.currentID = keyID
	return kp, nil
}

// PublicKey returns the public key from the enclave.
// The private key is NOT accessible — only signing operations are possible.
func (e *EnclaveKeyProvider) PublicKey() *rsa.PublicKey {
	e.mu.RLock()
	defer e.mu.RUnlock()

	pub, err := e.enclave.PublicKey(e.currentID)
	if err != nil {
		slog.Error("failed to get public key from enclave", "kid", e.currentID, "error", err)
		return nil
	}
	return pub
}

// PrivateKey returns nil — the private key is NOT available outside the enclave.
// All signing must go through Sign().
func (e *EnclaveKeyProvider) PrivateKey() *rsa.PrivateKey {
	// SECURITY: This method exists to satisfy the domain.KeyProvider interface.
	// In an enclave-backed system, the JWT signing code must be modified to
	// call enclave.Sign() instead of using jwt.NewWithClaims().Sign().
	// The private key must NEVER be materialized outside the enclave.
	return nil
}

// KeyID returns the current key identifier.
func (e *EnclaveKeyProvider) KeyID() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.currentID
}

// Sign delegates signing to the enclave. This replaces the standard
// jwt.NewWithClaims().Sign() pattern.
func (e *EnclaveKeyProvider) Sign(data []byte) ([]byte, error) {
	e.mu.RLock()
	kid := e.currentID
	e.mu.RUnlock()

	return e.enclave.Sign(kid, data)
}

// RotateKey generates a new key inside the enclave and seals it.
func (e *EnclaveKeyProvider) RotateKey() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	newID, err := e.enclave.RotateKey()
	if err != nil {
		return fmt.Errorf("enclave key rotation: %w", err)
	}

	e.previousID = e.currentID
	e.currentID = newID
	e.rotatedAt = time.Now()

	// Seal the new key for persistence.
	sealed, err := e.enclave.SealKey(newID)
	if err != nil {
		slog.Error("failed to seal rotated key", "kid", newID, "error", err)
		// Key is still in enclave memory; sealed copy is for persistence only.
	} else {
		// Persist sealed blob — caller should write to secure storage.
		// In production, this would be a write to an encrypted volume.
		slog.Info("rotated key sealed for persistence", "sealed_size", len(sealed))
	}

	slog.Info("enclave key rotated",
		"new_kid", e.currentID,
		"previous_kid", e.previousID,
		"grace_period", e.gracePeriod.String(),
	)
	return nil
}

// StartRotationTicker starts a background goroutine for periodic rotation.
func (e *EnclaveKeyProvider) StartRotationTicker(interval time.Duration) func() {
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := e.RotateKey(); err != nil {
					slog.Error("scheduled enclave key rotation failed", "error", err)
				}
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	return func() { close(done) }
}

// Attest generates an attestation report for remote verification.
func (e *EnclaveKeyProvider) Attest(challenge [64]byte) ([]byte, error) {
	return e.enclave.Attest(challenge)
}
```

### 4.4 Sealing Mechanism

Key sealing uses a key derived from the enclave's measurement:

```go
// EnclaveKeyDerivation shows how the sealing key is derived.
// In real SGX, this is done via EGETKEY with the SEAL_KEY request.
//
// The sealing key is derived from:
//   - MRENCLAVE (or MRSIGNER, depending on policy)
//   - CPU's root sealing key (embedded in hardware)
//
// This means:
//   - Only the same enclave (same code) can unseal the data
//   - If the enclave code changes (new version), old sealed data is lost
//   - A different physical machine cannot unseal (different CPU root key)

// SealingPolicyMRENCLAVE: key tied to exact enclave measurement.
// Pro: Strongest isolation — only identical code can unseal.
// Con: Key migration requires re-sealing after code updates.
//
// SealingPolicyMRSIGNER: key tied to enclave signer (developer key).
// Pro: Different versions of same developer's enclave can unseal.
// Con: Weaker isolation — any enclave signed by same key can access.

const (
	SealingPolicyMRENCLAVE = 0x0000
	SealingPolicyMRSIGNER  = 0x0001
)

// SealKeyInEnclave demonstrates the SGX sealing concept.
// The actual EGETKEY instruction is called inside C enclave code.
//
// In Go with EGo (edgeless/go), this is transparent — the runtime
// provides os.Seal() and os.Unseal() functions that wrap the
// underlying SGX instructions.
//
//	func sealKeyForPersistence(keyID string) ([]byte, error) {
//	    key := getEnclaveKey(keyID)
//	    plaintext := marshalPrivateKey(key)
//	    return os.Seal(plaintext) // SGX EGETKEY + AES-GCM
//	}
//
//	func unsealKey(sealed []byte) (*rsa.PrivateKey, error) {
//	    plaintext, err := os.Unseal(sealed) // SGX EGETKEY + AES-GCM verify
//	    if err != nil {
//	        return nil, fmt.Errorf("unseal failed (wrong enclave or tampered): %w", err)
//	    }
//	    return parsePrivateKey(plaintext)
//	}
```

### 4.5 Integration with Existing KeyProvider Interface

GGID's `domain.KeyProvider` interface returns `*rsa.PrivateKey`, which
assumes the key is available in process memory. For enclave-based key
management, this interface must be extended:

```go
// Current interface (services/oauth/internal/domain/models.go):
//
// type KeyProvider interface {
//     PublicKey() *rsa.PublicKey
//     PrivateKey() *rsa.PrivateKey
//     KeyID() string
// }

// Proposed extension for enclave support:
//
// type SigningKeyProvider interface {
//     PublicKey() *rsa.PublicKey
//     KeyID() string
//     // Sign signs the given data using the current key.
//     // For non-enclave providers, this delegates to PrivateKey().Sign().
//     // For enclave providers, this makes an ECALL to the enclave.
//     Sign(data []byte) ([]byte, error)
// }
//
// The OAuth service's token generation code would change from:
//   token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
//   token.Header["kid"] = kp.KeyID()
//   signed, err := token.SignedString(kp.PrivateKey())
//
// To:
//   header := map[string]any{"alg": "RS256", "typ": "JWT", "kid": kp.KeyID()}
//   payload, _ := json.Marshal(claims)
//   signingInput := base64(header) + "." + base64(payload)
//   sig, err := kp.Sign([]byte(signingInput))
//   token := signingInput + "." + base64(sig)
//
// This keeps the signing key inside the enclave and only exposes
// the signature to the host process.
```

---

## 5. Attestation Flow

### 5.1 Why Attestation Matters for IAM

Attestation is the cryptographic proof that your IAM service is running
in a genuine TEE. This enables:

1. **Client-side verification** — Users can verify that the auth service
   handling their password is running in a protected enclave.

2. **Service-to-service trust** — GGID's microservices can verify each
   other's enclave identity before exchanging tokens.

3. **Compliance evidence** — Auditors can verify that key management
   runs in hardware-protected memory.

4. **Secret provisioning** — A key management service can release keys
   *only* to an attested enclave, never to a plaintext process.

### 5.2 Full Attestation Flow

```
┌─────────────┐                  ┌──────────────┐                  ┌───────────────┐
│ GGID Auth   │                  │  Verifier /  │                  │  Key Release  │
│ (Enclave)   │                  │  Client      │                  │  Service      │
└──────┬──────┘                  └──────┬───────┘                  └───────┬───────┘
       │                                │                                  │
       │                                │  1. "I need to authenticate"     │
       │                                │ ◄──────────────────────────────  │
       │                                │                                  │
       │  2. Nonce (64 random bytes)    │                                  │
       │ ◄──────────────────────────────│                                  │
       │                                │                                  │
       │  3. EREPORT + EGETKEY          │                                  │
       │     Generate attestation quote │                                  │
       │     REPORT_DATA = nonce        │                                  │
       │                                │                                  │
       │  4. Quote + cert chain         │                                  │
       │ ──────────────────────────────►│                                  │
       │                                │                                  │
       │                                │  5. Verify:                      │
       │                                │     • Intel root CA → PCK → QE  │
       │                                │     • Quote signature valid     │
       │                                │     • MRENCLAVE == expected hash│
       │                                │     • REPORT_DATA == nonce      │
       │                                │     • TCB level acceptable       │
       │                                │                                  │
       │                                │  6. Attestation verified!       │
       │                                │     "This is a genuine enclave  │
       │                                │      running the expected code" │
       │                                │                                  │
       │                                │  7. Release signing key to      │
       │                                │     enclave via secure channel  │
       │                                │ ──────────────────────────────► │
       │                                │                                  │
       │  8. Encrypted key material     │                                  │
       │     (encrypted with enclave    │                                  │
       │      public key from report)   │                                  │
       │ ◄──────────────────────────────│────────────────────────────────  │
       │                                │                                  │
       │  9. Decrypt key inside enclave │                                  │
       │     Key is now in EPC memory   │                                  │
       │     Never visible to host OS   │                                  │
```

### 5.3 Quote Structure (SGX DCAP)

```go
// Quote represents an Intel SGX DCAP attestation quote.
// This is the raw structure generated by the Quoting Enclave.
type Quote struct {
	Header QuoteHeader
	Body   ReportBody   // The SGX report
	// Signature follows the body
}

type QuoteHeader struct {
	Version       uint16 // 3 for DCAP
	SignType      uint16 // 0 = unlinkable, 1 = linkable
	ISVSVN       [2]byte
	QESVN        [2]byte
	QEVendorID   [16]byte
	UserData     [20]byte
}

type ReportBody struct {
	CPUSVN       [16]byte     // CPU security version number
	MiscSelect   [4]byte
	ISVSVN       [2]byte      // Enclave security version
	MRSigner     [32]byte     // SHA-256 of enclave signing key
	MRENCLAVE    [32]byte     // SHA-256 of enclave code+data
	ReportData   [64]byte     // Caller-provided nonce/data
	Attributes   [16]byte     // Debug flag, mode, etc.
	Measurement  [32]byte
}

// The quote is signed by the Quoting Enclave (QE) using a certificate
// chain rooted in Intel's CA. The verifier must:
// 1. Extract the PCK (Platform Certificate Key) from the quote
// 2. Verify the PCK against Intel's root CA
// 3. Verify the QE's endorsement of the PCK
// 4. Verify the QE's signature on the quote
// 5. Check MRENCLAVE against the expected measurement
// 6. Check ReportData matches the expected nonce
```

### 5.4 Go Code: Attestation Verification

```go
// Package attestation provides verification of TEE attestation reports.
// This is used to verify that a remote service is running inside a
// genuine SGX/SEV-SNP enclave before releasing secrets or tokens.
package attestation

import (
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// ExpectedMeasurement is the MRENCLAVE hash of the trusted enclave code.
// This is computed during enclave build and pinned in the verifier.
// ANY change to the enclave code changes this hash.
type ExpectedMeasurement [32]byte

// AttestationVerifier verifies SGX DCAP attestation quotes.
type AttestationVerifier struct {
	// trustedMeasurements maps enclave names to their MRENCLAVE hashes.
	// Only enclaves with matching measurements are considered trustworthy.
	trustedMeasurements map[string]ExpectedMeasurement

	// intelRootCAs contains the Intel root CA certificates used to
	// verify the PCK certificate chain.
	intelRootCAs *x509.CertPool

	// minTCBLevel is the minimum Trusted Computing Base level accepted.
	minTCBLevel uint16

	// maxQuoteAge is the maximum acceptable age of an attestation quote.
	maxQuoteAge time.Duration
}

// NewAttestationVerifier creates a verifier with Intel root CAs.
func NewAttestationVerifier(rootCAs *x509.CertPool, minTCB uint16) *AttestationVerifier {
	return &AttestationVerifier{
		trustedMeasurements: make(map[string]ExpectedMeasurement),
		intelRootCAs:        rootCAs,
		minTCBLevel:         minTCB,
		maxQuoteAge:         5 * time.Minute,
	}
}

// RegisterEnclave registers an expected enclave measurement.
func (v *AttestationVerifier) RegisterEnclave(name string, mrenclave [32]byte) {
	v.trustedMeasurements[name] = ExpectedMeasurement(mrenclave)
}

// VerifyQuote verifies an SGX DCAP attestation quote.
//
// Parameters:
//   - quoteBytes: Raw quote bytes from the enclave
//   - expectedNonce: The nonce that was sent to the enclave (must match ReportData)
//   - expectedEnclaveName: The name of the expected enclave (looked up in trustedMeasurements)
//   - certChain: PCK certificate chain from the platform
//
// Returns nil if verification succeeds, error otherwise.
func (v *AttestationVerifier) VerifyQuote(
	quoteBytes []byte,
	expectedNonce [64]byte,
	expectedEnclaveName string,
	certChain []*x509.Certificate,
) error {
	if len(quoteBytes) < 432+64 { // Minimum quote size (header + body)
		return errors.New("quote too short")
	}

	// Parse the quote header.
	header, body, sig, err := parseQuoteStructure(quoteBytes)
	if err != nil {
		return fmt.Errorf("parse quote: %w", err)
	}

	// Step 1: Verify the certificate chain against Intel root CA.
	if err := verifyCertChain(certChain, v.intelRootCAs); err != nil {
		return fmt.Errorf("cert chain verification failed: %w", err)
	}

	// Step 2: Verify the quote signature.
	if err := verifyQuoteSignature(quoteBytes[:432], sig, certChain[0].PublicKey); err != nil {
		return errors.New("quote signature verification failed")
	}

	// Step 3: Check TCB (Trusted Computing Base) level.
	tcbLevel := binary.LittleEndian.Uint16(header.QESVN[:])
	if tcbLevel < v.minTCBLevel {
		return fmt.Errorf("TCB level %d below minimum %d", tcbLevel, v.minTCBLevel)
	}

	// Step 4: Verify MRENCLAVE matches expected enclave.
	expectedMrenclave, ok := v.trustedMeasurements[expectedEnclaveName]
	if !ok {
		return fmt.Errorf("unknown enclave: %s", expectedEnclaveName)
	}

	var actualMrenclave [32]byte
	copy(actualMrenclave[:], body.MRENCLAVE[:])

	if actualMrenclave != expectedMrenclave {
		return fmt.Errorf("MRENCLAVE mismatch: expected %s, got %s",
			hex.EncodeToString(expectedMrenclave[:]),
			hex.EncodeToString(actualMrenclave[:]),
		)
	}

	// Step 5: Verify the nonce (ReportData) matches what we sent.
	var reportData [64]byte
	copy(reportData[:], body.ReportData[:])

	if reportData != expectedNonce {
		return errors.New("nonce mismatch in ReportData — possible replay attack")
	}

	// Step 6: Check quote freshness (if timestamp is embedded).
	// In production, this would check the timestamp embedded in the
	// QE report or use a separate timestamp authority.

	// Step 7: Verify debug flag is not set (production enclaves should
	// not be debuggable).
	if body.Attributes[0]&0x02 != 0 {
		return errors.New("enclave is in debug mode — not suitable for production")
	}

	return nil
}

// parseQuoteStructure splits the raw quote into header, body, and signature.
func parseQuoteStructure(raw []byte) (QuoteHeader, ReportBody, []byte, error) {
	// DCAP quote layout:
	//   Bytes 0-47:    QuoteHeader (48 bytes)
	//   Bytes 48-431:  ReportBody (384 bytes)
	//   Bytes 432+:    Signature (variable length)

	if len(raw) < 48+384 {
		return QuoteHeader{}, ReportBody{}, nil, errors.New("quote too short for header + body")
	}

	var header QuoteHeader
	if err := binary.Read(newBytesReader(raw[:48]), binary.LittleEndian, &header); err != nil {
		return QuoteHeader{}, ReportBody{}, nil, fmt.Errorf("parse header: %w", err)
	}

	var body ReportBody
	if err := binary.Read(newBytesReader(raw[48:432]), binary.LittleEndian, &body); err != nil {
		return QuoteHeader{}, ReportBody{}, nil, fmt.Errorf("parse body: %w", err)
	}

	signature := raw[432:]
	return header, body, signature, nil
}

// verifyCertChain verifies the PCK certificate chain against Intel root CAs.
func verifyCertChain(chain []*x509.Certificates, roots *x509.CertPool) error {
	if len(chain) == 0 {
		return errors.New("empty certificate chain")
	}

	// Verify the leaf certificate (PCK) against the chain.
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: x509.NewCertPool(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	// Add intermediates (QE cert, Intel CA intermediates).
	for i := 1; i < len(chain); i++ {
		opts.Intermediates.AddCert(chain[i])
	}

	_, err := chain[0].Verify(opts)
	return err
}

// verifyQuoteSignature verifies the ECDSA signature on the quote.
func verifyQuoteSignature(quoteBody []byte, sig []byte, pubKey any) error {
	// Implementation depends on the quote signature type (ECDSA-256-with-SHA256).
	// Use crypto/ecdsa.VerifyASN1 for the actual verification.
	// Skipped here for brevity.
	return nil
}

// newBytesReader creates a bytes.Reader (avoiding bytes import in example).
func newBytesReader(b []byte) *bytesReader { return &bytesReader{data: b} }

type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, errors.New("EOF")
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
```

### 5.5 Attestation for SEV-SNP

For SEV-SNP, the attestation flow is similar but uses AMD's VCEK
certificate chain instead of Intel's:

```go
// SEVSNPAttestationVerifier verifies AMD SEV-SNP attestation reports.
type SEVSNPAttestationVerifier struct {
	// amdRootCAs contain AMD's ARK (AMD Root Key) certificates.
	amdRootCAs *x509.CertPool

	// expectedMeasurement is the expected launch measurement of the VM.
	expectedMeasurement [48]byte

	// expectedFamilyID and expectedImageID identify the workload.
	expectedFamilyID [16]byte
	expectedImageID  [16]byte
}

// SNPReport represents the SEV-SNP attestation report structure.
type SNPReport struct {
	Version          uint32
	LaunchSVN        uint8
	Policy           SNPReportPolicy
	FamilyID         [16]byte
	ImageID          [16]byte
	VMPL             uint32
	SignatureAlgo    uint32
	PlatformVersion  SNPPlatformVersion
	PlatformInfo     uint8
	Flags            SNPReportFlags
	ReportData       [64]byte  // Caller-provided nonce
	Measurement      [48]byte  // Hash of VM initial state
	HostData         [32]byte
	IDKeyDigest      [48]byte
	AuthorKeyDigest  [48]byte
	ReportID         [32]byte
	ReportIDMA       [32]byte
	ReportTCB        uint64
	// ... followed by signature
}

func (v *SEVSNPAttestationVerifier) Verify(report *SNPReport, vcekCert *x509.Certificate, expectedNonce [64]byte) error {
	// 1. Verify VCEK certificate chain against AMD ARK.
	if err := verifyAMDChain(vcekCert, v.amdRootCAs); err != nil {
		return fmt.Errorf("VCEK chain verification: %w", err)
	}

	// 2. Verify report signature using VCEK public key.
	if err := verifySNPSignature(report, vcekCert.PublicKey); err != nil {
		return errors.New("SEV-SNP report signature invalid")
	}

	// 3. Verify nonce.
	if report.ReportData != expectedNonce {
		return errors.New("nonce mismatch")
	}

	// 4. Verify VM measurement.
	if report.Measurement != v.expectedMeasurement {
		return fmt.Errorf("measurement mismatch: expected %x, got %x",
			v.expectedMeasurement, report.Measurement)
	}

	// 5. Verify FamilyID and ImageID.
	if report.FamilyID != v.expectedFamilyID {
		return errors.New("family ID mismatch")
	}
	if report.ImageID != v.expectedImageID {
		return errors.New("image ID mismatch")
	}

	// 6. Verify TCB version is acceptable.
	// ...

	return nil
}
```

---

## 6. Go SGX/TEE SDK Landscape

### 6.1 Ecosystem Overview

Running Go inside a TEE is challenging because Go's runtime, garbage
collector, and standard library were not designed for enclave
constraints (no direct I/O, limited memory, no fork). Several
projects address this:

```
┌──────────────────────────────────────────────────────────────────┐
│                Go TEE SDK Landscape                               │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │                    Intel SGX                               │    │
│  │                                                           │    │
│  │  EGo (edgeless/go) ◄── Best Go SGX option               │    │
│  │  ├─ Patched Go runtime that runs inside SGX              │    │
│  │  ├─ Automatic ECALL/OCALL bridging                       │    │
│  │  ├─ os.Seal()/os.Unseal() for key sealing                │    │
│  │  ├─ Remote attestation library                           │    │
│  │  └─ Open source (BSD-3-Clause)                           │    │
│  │                                                           │    │
│  │  Gramine (formerly Graphene-SGX)                         │    │
│  │  ├─ Library OS — runs unmodified Linux apps in SGX       │    │
│  │  ├─ Supports Go via syscall interception                 │    │
│  │  ├─ No code changes needed                               │    │
│  │  ├─ Performance overhead from syscall translation        │    │
│  │  └─ Linux Foundation project                             │    │
│  │                                                           │    │
│  │  Occlum                                                   │    │
│  │  ├─ Library OS for SGX (Rust-based)                      │    │
│  │  ├─ Supports multi-process apps in enclave               │    │
│  │  ├─ File system, networking, threads inside enclave      │    │
│  │  ├─ Good Go support                                      │    │
│  │  └─ Open source (BSD-style)                              │    │
│  │                                                           │    │
│  │  Fortanix EDP (Rust, not Go)                             │    │
│  │  ├─ Rust SDK for SGX                                     │    │
│  │  ├─ Most mature enclave SDK                              │    │
│  │  └─ No Go support                                        │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │              AMD SEV-SNP / Confidential VMs               │    │
│  │                                                           │    │
│  │  Confidential VMs                                          │    │
│  │  ├─ Azure Confidential VM (DCasv5/ECasv5 series)         │    │
│  │  ├─ AWS Nitro Enclaves (similar concept, different HW)   │    │
│  │  ├─ GCP Confidential VMs (SEV-SNP on AMD)                │    │
│  │  ├─ NO code changes — entire VM is encrypted              │    │
│  │  └─ Go apps run unmodified                                │    │
│  │                                                           │    │
│  │  Confidential Containers (Kubernetes)                    │    │
│  │  ├─ Kata Containers with hardware TEE support             │    │
│  │  ├─ Pod annotation: io.katacontainers.confidential        │    │
│  │  ├─ Attestation via attestation-agent sidecar             │    │
│  │  └─ CC Consortium standardization                         │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │              NVIDIA Confidential Computing (GPU)           │    │
│  │                                                           │    │
│  │  ├─ Hopper H100: confidential computing on GPU            │    │
│  │  ├─ Useful for ML-based IAM anomaly detection             │    │
│  │  └─ Not directly relevant for key management              │    │
│  └──────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

### 6.2 EGo (edgeless/go)

EGo is the most promising option for running Go code inside SGX:

```go
// Example: Running GGID's key management inside SGX using EGo
//
// EGo provides a patched Go runtime that:
// 1. Runs inside the SGX enclave
// 2. Intercepts syscalls and translates them to OCALLs
// 3. Provides os.Seal()/os.Unseal() for enclave key sealing
// 4. Handles remote attestation via its spin/attestation package

// Build: ego-go build -o ggid-auth auth_service.go
// Sign:  ego sign ggid-auth  (generates enclave signature)
// Run:   ego run ggid-auth

package main

import (
	"crypto/rsa"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/edgelesssys/ego/attestation"
)

// enclaveKeyStore holds the RSA signing key inside the SGX enclave.
// This struct exists ONLY inside enclave memory (EPC).
var enclaveKeyStore struct {
	signingKey *rsa.PrivateKey
	keyID      string
}

func init() {
	// Try to unseal existing key.
	if sealedData, err := os.ReadFile("/var/lib/ggid/sealed_key.bin"); err == nil {
		// os.Unseal decrypts using a key derived from MRENCLAVE.
		// This only works if the enclave code hasn't changed.
		plaintext, err := os.Unseal(sealedData)
		if err == nil {
			// Parse and load the key.
			key, err := parsePKCS1PrivateKey(plaintext)
			if err == nil {
				enclaveKeyStore.signingKey = key
				enclaveKeyStore.keyID = computeKID(&key.PublicKey)
				log.Println("signing key unsealed from disk")
				return
			}
		}
	}

	// Generate new key inside enclave.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal("failed to generate enclave key:", err)
	}

	enclaveKeyStore.signingKey = key
	enclaveKeyStore.keyID = computeKID(&key.PublicKey)

	// Seal and persist.
	plaintext := marshalPKCS1PrivateKey(key)
	sealed, err := os.Seal(plaintext)
	if err != nil {
		log.Fatal("failed to seal enclave key:", err)
	}

	if err := os.WriteFile("/var/lib/ggid/sealed_key.bin", sealed, 0600); err != nil {
		log.Fatal("failed to persist sealed key:", err)
	}

	log.Println("new signing key generated and sealed inside enclave")
}

// signJWT signs a JWT inside the enclave. The signing key never
// leaves EPC memory.
func signJWT(claims map[string]any) (string, error) {
	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": enclaveKeyStore.keyID,
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	signingInput := base64URLEncode(headerJSON) + "." + base64URLEncode(claimsJSON)

	hash := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, enclaveKeyStore.signingKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	return signingInput + "." + base64URLEncode(signature), nil
}

// attestHandler generates an SGX attestation report.
func attestHandler(w http.ResponseWriter, r *http.Request) {
	// Generate attestation report.
	// EGo's attestation.Generate() creates an SGX quote with the
	// provided report data (nonce).
	reportData := generateNonce()

	attestationDoc, err := attestation.Generate(reportData)
	if err != nil {
		http.Error(w, "attestation failed", http.StatusInternalServerError)
		return
	}

	// Return the attestation quote to the client for verification.
	json.NewEncoder(w).Encode(map[string]any{
		"quote":     attestationDoc,
		"nonce":     reportData,
		"enclave":   "ggid-auth-sgx",
		"key_id":    enclaveKeyStore.keyID,
	})
}

func main() {
	http.HandleFunc("/attest", attestHandler)
	http.HandleFunc("/token", tokenHandler)
	log.Println("GGID Auth running inside SGX enclave")
	log.Fatal(http.ListenAndServe(":9001", nil))
}
```

### 6.3 EGo Limitations for Go

EGo has several constraints that affect GGID:

| Limitation | Impact on GGID | Mitigation |
|---|---|---|
| **No `cgo`** | No C library dependencies | Pure-Go alternatives needed |
| **No `os/exec`** | Cannot spawn processes | OK — GGID doesn't exec |
| **Limited file system** | SGX has no real FS | EGo provides virtual FS |
| **EPC size** | Argon2id 64MB may not fit | Reduce memory cost in enclave |
| **No `unsafe` in enclave code** | Some crypto libs use unsafe | Use stdlib crypto only |
| **No `net.Listen` optimization** | Networking via OCALL | Acceptable for IAM throughput |
| **Single-threaded GC** | GC pauses in enclave | Reduce heap, tune GOGC |

### 6.4 Confidential Containers

For Kubernetes-based deployments, Confidential Containers offer the
simplest path to confidential computing for GGID:

```yaml
# Example: GGID Auth deployment with Confidential Containers
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ggid-auth-confidential
  namespace: ggid
spec:
  replicas: 3
  selector:
    matchLabels:
      app: ggid-auth
  template:
    metadata:
      labels:
        app: ggid-auth
      annotations:
        # Enable confidential computing via Kata Containers
        io.katacontainers.confidential: "true"
        # Require SEV-SNP hardware
        io.katacontainers.require-attestation: "true"
    spec:
      runtimeClassName: kata-qemu-sev  # SEV-SNP runtime
      containers:
        - name: auth
          image: ggid/auth:latest
          ports:
            - containerPort: 9001
          env:
            - name: SGX_ENABLED
              value: "true"
            - name: ENCLAVE_KEY_PATH
              value: "/var/lib/ggid/sealed_key.bin"
          resources:
            requests:
              memory: "512Mi"
              cpu: "500m"
            limits:
              memory: "1Gi"
              cpu: "1000m"
      nodeSelector:
        # Schedule on SEV-SNP-capable nodes
        confidentialcomputing.azure.com/sgx: "true"
```

### 6.5 Gramine for Unmodified Go

Gramine can run unmodified Go binaries inside SGX via its library OS:

```ini
# Gramine manifest for GGID Auth service (ggid-auth.manifest)
libos.entrypoint = "/usr/local/bin/ggid-auth"

# SGX configuration
sgx.enclave_size = "1073741824"  # 1 GB EPC (with oversubscription)
sgx.thread_num = 16
sgx.isvprodid = 1
sgx.isvsvn = 1

# Allowed files and directories
fs.mounts = [
  { path = "/etc/ggid", uri = "file:/etc/ggid" },
  { path = "/var/lib/ggid", uri = "file:/var/lib/ggid" },
  { path = "/tmp", uri = "file:/tmp" },
]

# Network access
net.rules = [{protocol = "tcp", port = "9001"}]

# Environment variables
env.GID_LISTEN_ADDR = "0.0.0.0:9001"
env.GID_DB_HOST = "postgres.ggid.svc.cluster.local"
env.GID_ENCLAVE_MODE = "gramine"

# Key sealing
loader.env.LD_LIBRARY_PATH = "/lib:/usr/lib"
sgx.remote_attestation = "dcap"
```

### 6.6 AWS Nitro Enclaves

AWS Nitro Enclaves are NOT based on SGX or SEV — they use the Nitro
Hypervisor with hardware isolation via AWS's custom silicon. They offer:

- No specific hardware requirement (works on all EC7 instances)
- Linux-only VM isolated from parent instance
- vsock communication channel (no network)
- AWS KMS integration for key release via attestation document
- **Limitation:** No CPU-level memory encryption (relies on hypervisor isolation)

For GGID on AWS, Nitro Enclaves could run the JWT signing component
with KMS-released keys, but without CPU-level memory encryption, it
does not provide the same guarantees as SGX/SEV-SNP.

---

## 7. Performance Overhead Analysis

### 7.1 SGX Enclave Transition Cost

Every call from untrusted code into the enclave (`ECALL`) and back
(`OCALL`) incurs overhead:

```
┌───────────────────────────────────────────────────────────────┐
│                   ECALL/OCALL Cost Breakdown                   │
│                                                                │
│  Operation                     Cycles    Time (3.0 GHz CPU)   │
│  ──────────────────────────────────────────────────────────  │
│  ECALL entry (EENTER)         ~2,000      ~0.67 μs            │
│  Save/restore registers       ~1,000      ~0.33 μs            │
│  TLB flush (AEX handling)     ~2,000      ~0.67 μs            │
│  OCALL exit (EEXIT)           ~1,500      ~0.50 μs            │
│  ECALL return                 ~1,500      ~0.50 μs            │
│  ──────────────────────────────────────────────────────────  │
│  Total round-trip            ~8,000       ~2.67 μs            │
│                                                                │
│  Compare: Normal function call ~10 cycles ~0.003 μs            │
│  ECALL is ~800x slower than a normal function call             │
└───────────────────────────────────────────────────────────────┘
```

For JWT signing, this means:
- RSA-2048 sign in native Go: ~1 ms
- RSA-2048 sign in SGX enclave: ~1 ms + ~3 μs ECALL overhead ≈ 1.003 ms
- The ECALL overhead is negligible for RSA signing (<0.3%)

But for operations that require many enclave transitions (e.g., per-field
PII encryption), the overhead accumulates:
- 100 field encryptions × 3 μs ECALL = 0.3 ms overhead (significant if
  each AES-GCM encryption is <1 μs)

### 7.2 Memory Encryption Overhead

The MEE (Memory Encryption Engine) adds overhead to every memory access
within the EPC:

| Operation | Native | SGX Enclave | Overhead |
|---|---|---|---|
| Sequential memory read (1 MB) | 0.31 ms | 0.33 ms | +6.5% |
| Random memory read (1 MB) | 1.20 ms | 1.28 ms | +6.7% |
| SHA-256 hash (1 MB) | 0.45 ms | 0.47 ms | +4.4% |
| AES-256-GCM encrypt (1 MB) | 0.12 ms | 0.13 ms | +8.3% |
| RSA-2048 sign | 1.02 ms | 1.08 ms | +5.9% |
| RSA-2048 verify | 0.03 ms | 0.04 ms | +33% (small absolute) |
| HMAC-SHA256 (100 bytes) | 0.001 ms | 0.002 ms | +100% (ECALL dominates) |

### 7.3 EPC Swap Overhead

When enclave memory exceeds EPC capacity, the CPU must swap pages in/out
of the EPC. This is catastrophic for performance:

```
┌──────────────────────────────────────────────────────────┐
│              EPC Swap Performance Impact                  │
│                                                          │
│  EPC Usage    Swap Behavior        Performance Impact     │
│  ────────────────────────────────────────────────────── │
│  < 90%        No swapping          Baseline               │
│  90-95%       Occasional swaps     ~2x slower             │
│  95-100%      Frequent swapping    ~10-100x slower        │
│  > 100% (SGX2 oversubscription)    Hardware-assisted      │
│                (dynamic EPC)       ~2-5x slower           │
│                                                          │
│  GGID impact:                                            │
│  Argon2id with 64 MB memory cost + Go runtime (~30 MB)   │
│  + service code (~20 MB) = ~114 MB total EPC usage       │
│  → EXCEEDS 128 MB SGX1 EPC limit                         │
│  → Will cause severe swapping on SGX1                     │
│  → Must use SGX2 with larger EPC or reduce Argon2id      │
└──────────────────────────────────────────────────────────┘
```

### 7.4 Benchmark: Go Service in SGX vs Native

Estimated performance for GGID's key operations:

```go
// Benchmark results (estimated, based on published SGX benchmarks)
//
// BenchmarkNative_JWTSign-8        1000   1024000 ns/op   # Native Go
// BenchmarkSGX_JWTSign-8            950   1078000 ns/op   # EGo SGX
//                                              +5.3% overhead
//
// BenchmarkNative_PasswordHash-8      5  256000000 ns/op   # Native Argon2id 64MB
// BenchmarkSGX_PasswordHash-8         2  850000000 ns/op   # EGo SGX (EPC swap!)
//                                             +232% overhead (EPC swapping)
//
// BenchmarkSGX2_PasswordHash-8        4  289000000 ns/op   # SGX2 with 512MB EPC
//                                              +12.9% overhead (acceptable)
//
// BenchmarkNative_AES256GCM-8     10000     120000 ns/op   # Native AES-GCM
// BenchmarkSGX_AES256GCM-8         9200     130000 ns/op   # SGX AES-GCM
//                                              +8.3% overhead
//
// BenchmarkNative_RSAVerify-8    100000      30000 ns/op   # Native RSA verify
// BenchmarkSGX_RSAVerify-8        75000      40000 ns/op   # SGX RSA verify
//                                             +33% overhead (small absolute)
```

### 7.5 AMD SEV-SNP Overhead

SEV-SNP has lower overhead than SGX because it operates at the VM level
with fewer transitions:

| Operation | Native VM | SEV-SNP VM | Overhead |
|---|---|---|---|
| Memory access (sequential) | 0.31 ms | 0.32 ms | +3.2% |
| Memory access (random) | 1.20 ms | 1.26 ms | +5.0% |
| Network I/O (10 KB) | 0.15 ms | 0.16 ms | +6.7% |
| Disk I/O (10 KB) | 0.08 ms | 0.08 ms | ~0% |
| RSA-2048 sign | 1.02 ms | 1.05 ms | +2.9% |
| Full HTTP request | 2.50 ms | 2.60 ms | +4.0% |

### 7.6 Performance vs Security Tradeoff

```
┌────────────────────────────────────────────────────────────────┐
│              Performance vs Security Tradeoff Matrix            │
│                                                                 │
│  Security Level    Technology    Overhead    Deployment Cost    │
│  ────────────────────────────────────────────────────────────  │
│  Level 0: None     Plaintext      0%         $0                │
│  (current GGID)    in memory                                                  │
│                                                                 │
│  Level 1: Env-     Env vars +     0%         $0                │
│  Vars + Pepper     Argon2id                                                    │
│                                                                 │
│  Level 2: HSM      PKCS#11 to     ~1-5 ms   $5K-$50K/HSM      │
│  (key storage)     network HSM    per crypto   + network       │
│                    for signing    operation    latency         │
│                                                                 │
│  Level 3: CVM      SEV-SNP VM     ~3-5%      ~10-25% cloud     │
│  (VM encryption)   for entire     overhead    instance cost    │
│                    service         surcharge                   │
│                                                                 │
│  Level 4: SGX      EGo enclave    ~5-10%     SGX-capable      │
│  (app enclave)     for key ops    overhead    cloud instances   │
│                    (native       +EPC swap    (~20-50% more)   │
│                    perf if EPC   overhead     + refactoring    │
│                    is large enough)            effort          │
│                                                                 │
│  Level 5: SGX +    Full enclave   ~10-20%    Maximum effort   │
│  HSM (defense      for app + HSM  overhead    + maximum cost   │
│  in depth)         for root keys                                 │
│                                                                 │
│  Recommended for GGID: Level 3 (CVM) as baseline,               │
│  Level 4 (SGX) for Auth/OAuth key operations,                   │
│  Level 5 (SGX+HSM) for compliance-driven deployments.           │
└────────────────────────────────────────────────────────────────┘
```

---

## 8. When to Use Confidential Computing vs HSM

### 8.1 HSM Strengths

HSMs excel at:
- **Key custody** — Keys generated inside HSM, never exportable
- **FIPS 140-2/140-3 certification** — Required for FedRAMP, PCI-DSS, FISMA
- **Physical tamper resistance** — Active zeroization on tamper detection
- **High-throughput crypto** — Hardware acceleration for RSA/ECDSA operations
- **Audit trail** — All key usage logged inside HSM
- **Dual control** — M-of-N quorum for key operations

### 8.2 Confidential Computing Strengths

TEE-based approaches excel at:
- **Protecting application logic** — Entire authentication flow is protected
- **Protecting PII in memory** — User emails, phone numbers, passwords
- **Protecting session state** — Active sessions, tokens, authorization codes
- **Attestable runtime** — Cryptographic proof of what code is running
- **No hardware procurement** — Available as cloud instances
- **Programmable** — Custom logic runs inside protected memory

### 8.3 What Each Cannot Do

```
┌─────────────────────────────────────────────────────────────┐
│              What HSMs and TEEs Cannot Do                    │
│                                                             │
│  HSM CANNOT:                                               │
│  ✗ Protect application memory (PII is in plaintext)        │
│  ✗ Protect application logic (auth flows are visible)      │
│  ✗ Prevent memory dumps of session data                    │
│  ✗ Protect against process-level attacks (ptrace, etc.)    │
│  ✗ Scale elastically (physical device)                     │
│                                                             │
│  TEE CANNOT:                                               │
│  ✗ Match HSM physical tamper resistance                    │
│  ✗ Provide FIPS 140-2/3 certification                      │
│  ✗ Protect against CPU side-channel attacks (all of them)  │
│  ✗ Protect against firmware/hardware bugs                  │
│  ✗ Guarantee key survival across CPU replacement           │
│  ✗ Prevent denial of service (host controls scheduling)    │
└─────────────────────────────────────────────────────────────┘
```

### 8.4 Decision Matrix

| Requirement | HSM | TEE (SGX/SEV) | Both |
|---|---|---|---|
| Protect JWT signing key from extraction | Yes | Yes | Yes |
| Protect PII in application memory | No | **Yes** | Yes |
| Protect authentication logic from inspection | No | **Yes** | Yes |
| FIPS 140-2 certification required | **Yes** | No | Yes |
| Protect against physical tampering | **Yes** | No | Yes |
| Protect against memory dumps | No | **Yes** | Yes |
| Attestable runtime proof | Limited | **Yes** | Yes |
| Minimal code changes needed | Moderate | Varies | High |
| Cloud-native elastic scaling | Via KMS | **Yes** | Yes |
| Cost-effective for startup | Via cloud KMS | **Yes** | Expensive |
| Regulatory compliance (FedRAMP High) | **Required** | Helps | **Required** |

### 8.5 GGID Recommendation

For GGID specifically:

1. **Phase 1 (Now):** Use cloud KMS (AWS KMS, GCP KMS) for key storage.
   This is the existing HSM recommendation from `hsm-kms-integration.md`.

2. **Phase 2 (Near-term):** Deploy GGID services on Confidential VMs
   (Azure DCsv3, GCP SEV-SNP instances). Zero code changes, protects all
   PII in memory.

3. **Phase 3 (When needed):** Migrate JWT signing to SGX enclave via EGo.
   Protects signing key even from memory dumps.

4. **Phase 4 (Compliance):** Add HSM for root CA and compliance-required
   key operations. Use SGX for application-level protection.

---

## 9. Confidential Computing for Multi-Tenant IAM

### 9.1 The Multi-Tenant Confidentiality Problem

GGID is a multi-tenant IAM platform. In a SaaS deployment, multiple
tenants share the same infrastructure. The cloud operator has access
to all tenant data.

**Threat model:**

```
┌────────────────────────────────────────────────────────────────┐
│           Multi-Tenant IAM Threat Model                         │
│                                                                │
│  Without Confidential Computing:                               │
│                                                                │
│  ┌─────────────────────────────────────────────────┐          │
│  │              Cloud Operator / Admin              │          │
│  │  Can access:                                    │          │
│  │  • Tenant A's user database (via DB access)     │          │
│  │  • Tenant B's JWT signing keys (via VM memory)  │          │
│  │  • Tenant C's OAuth secrets (via disk snapshot) │          │
│  │  • All tenants' audit logs                      │          │
│  │  • Inter-service gRPC traffic (via hypervisor)  │          │
│  └─────────────────────────────────────────────────┘          │
│                                                                │
│  With Confidential VMs:                                        │
│                                                                │
│  ┌─────────────────────────────────────────────────┐          │
│  │              Cloud Operator / Admin              │          │
│  │  Can access:                                    │          │
│  │  • Encrypted memory only (ciphertext)           │          │
│  │  • Cannot read tenant data even with root       │          │
│  │  • Cannot extract signing keys                  │          │
│  │  • Can only affect availability (DoS)           │          │
│  │  Cannot access:                                 │          │
│  │  ✗ Tenant data in memory                        │          │
│  │  ✗ JWT signing keys                             │          │
│  │  ✗ OAuth secrets                                │          │
│  │  ✗ Audit log contents                           │          │
│  └─────────────────────────────────────────────────┘          │
└────────────────────────────────────────────────────────────────┘
```

### 9.2 Per-Tenant Enclave Isolation

For maximum isolation, each tenant's data can be processed by a separate
enclave:

```
┌──────────────────────────────────────────────────────────────┐
│            Per-Tenant Enclave Isolation Architecture          │
│                                                               │
│                     API Gateway (untrusted)                   │
│                          │                                    │
│              ┌───────────┼───────────┐                       │
│              │           │           │                        │
│              ▼           ▼           ▼                        │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐         │
│  │ Tenant A      │ │ Tenant B      │ │ Tenant C      │         │
│  │ Enclave       │ │ Enclave       │ │ Enclave       │         │
│  │               │ │               │ │               │         │
│  │ Key: K_A      │ │ Key: K_B      │ │ Key: K_C      │         │
│  │ PII: {A users}│ │ PII: {B users}│ │ PII: {C users}│         │
│  │ DB: pgcrypto  │ │ DB: pgcrypto  │ │ DB: pgcrypto  │         │
│  │  with tenant   │ │  with tenant   │ │  with tenant   │         │
│  │  encryption    │ │  encryption    │ │  encryption    │         │
│  │  key           │ │  key           │ │  key           │         │
│  │               │ │               │ │               │         │
│  │ EPC isolated  │ │ EPC isolated  │ │ EPC isolated  │         │
│  │ from B and C  │ │ from A and C  │ │ from A and B  │         │
│  └──────────────┘ └──────────────┘ └──────────────┘         │
│                                                               │
│  Each enclave has:                                            │
│  • Separate signing key (K_A, K_B, K_C)                      │
│  • Separate PII namespace                                     │
│  • Separate database encryption key                           │
│  • Separate attestation identity                              │
│                                                               │
│  Cross-tenant memory access: IMPOSSIBLE (hardware enforced)   │
└──────────────────────────────────────────────────────────────┘
```

### 9.3 Confidential VMs for Tenant-Specific Deployments

For high-value tenants (government, financial), GGID can deploy
dedicated Confidential VMs:

```yaml
# Kubernetes deployment: Dedicated Confidential VM per tenant
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ggid-auth-tenant-acme-corp
  namespace: ggid-isolated
  labels:
    tenant: "acme-corp"
    confidential: "true"
spec:
  replicas: 2
  template:
    metadata:
      annotations:
        io.katacontainers.confidential: "true"
        # Tenant-specific enclave measurement
        io.ggid.enclave-measurement: "a3f5e8c1..."
    spec:
      runtimeClassName: kata-qemu-sev
      nodeSelector:
        confidentialcomputing.io/sev-snp: "true"
        # Dedicated node pool for this tenant
        tenant-pool: "acme-corp"
      containers:
        - name: auth
          image: ggid/auth:latest
          env:
            - name: TENANT_ID
              value: "acme-corp-uuid"
            - name: CONFIDENTIAL_MODE
              value: "sev-snp"
            - name: ENCLAVE_KEY_RELEASE_URL
              # Keys released only after attestation verification
              value: "https://kms.ggid.dev/v1/release"
```

### 9.4 Attestation-Based Tenant Verification

Tenants can verify that their data is processed in a genuine enclave:

```go
// Package tenantattest provides tenant-side attestation verification.
// A tenant (or their auditor) can verify that GGID is running their
// tenant's authentication service inside a genuine TEE.
package tenantattest

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TenantAttestationClient allows a tenant to verify GGID's enclave.
type TenantAttestationClient struct {
	ggidAuthURL       string
	expectedMRENCLAVE [32]byte
	httpClient        *http.Client
}

// VerifyGGIDEnclave verifies that the GGID Auth service is running
// inside a genuine SGX enclave with the expected code measurement.
//
// This gives tenants cryptographic assurance that:
// 1. Their data is processed in encrypted memory
// 2. The GGID code has not been tampered with
// 3. Cloud operators cannot read tenant data
func (c *TenantAttestationClient) VerifyGGIDEnclave() error {
	// Step 1: Generate nonce.
	nonce := make([]byte, 64)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("generate nonce: %w", err)
	}

	// Step 2: Request attestation from GGID Auth.
	reqBody, _ := json.Marshal(map[string]any{
		"nonce": nonce,
	})
	resp, err := c.httpClient.Post(
		c.ggidAuthURL+"/.well-known/enclave-attestation",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return fmt.Errorf("request attestation: %w", err)
	}
	defer resp.Body.Close()

	var attResp struct {
		Quote       []byte            `json:"quote"`
		CertChain   [][]byte          `json:"cert_chain"`
		EnclaveInfo map[string]string `json:"enclave_info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&attResp); err != nil {
		return fmt.Errorf("decode attestation response: %w", err)
	}

	// Step 3: Verify the quote.
	verifier := NewAttestationVerifier(
		loadIntelRootCAs(),
		MinimumTCBLevel,
	)
	verifier.RegisterEnclave("ggid-auth", c.expectedMRENCLAVE)

	var nonceArray [64]byte
	copy(nonceArray[:], nonce)

	if err := verifier.VerifyQuote(
		attResp.Quote,
		nonceArray,
		"ggid-auth",
		parseCertChain(attResp.CertChain),
	); err != nil {
		return fmt.Errorf("attestation verification failed: %w", err)
	}

	// Step 4: Verify freshness.
	reportTime := attResp.EnclaveInfo["timestamp"]
	if isStale(reportTime, 5*time.Minute) {
		return errors.New("attestation report is stale")
	}

	return nil
}
```

---

## 10. Remote Attestation Integration

### 10.1 Attested Service-to-Service Authentication

In a zero-trust architecture, GGID's microservices should verify each
other's enclave identity. This creates a hardware-rooted service mesh:

```
┌──────────────────────────────────────────────────────────────────┐
│         GGID Attested Service Mesh Architecture                    │
│                                                                   │
│  ┌──────────┐  attest  ┌───────────┐  attest  ┌──────────┐       │
│  │ Gateway  │ ◄──────► │  Auth     │ ◄──────► │ Identity │       │
│  │ (TEE)    │  verify  │ (TEE)     │  verify  │ (TEE)    │       │
│  └────┬─────┘          └─────┬─────┘          └────┬─────┘       │
│       │                      │                      │             │
│       │    ┌─────────────────┼──────────────────────┘             │
│       │    │                 │                                    │
│       │    ▼                 ▼                                    │
│  ┌────┴────────────────────────────────────────────────────┐     │
│  │              Attestation Authority (AA)                   │     │
│  │                                                          │     │
│  │  • Verifies enclave quotes from all services              │     │
│  │  • Issues short-lived attestation tokens                  │     │
│  │  • Revokes compromised enclaves                           │     │
│  │  • Maintains trusted MRENCLAVE registry                   │     │
│  └──────────────────────────────────────────────────────────┘     │
│                                                                   │
│  Token Issuance Flow:                                             │
│                                                                   │
│  1. Service A starts in TEE → generates attestation quote         │
│  2. Service A → AA: "Here is my quote"                            │
│  3. AA verifies quote → issues attestation JWT (5 min TTL)        │
│  4. Service A → Service B: request + attestation JWT              │
│  5. Service B verifies JWT → checks attestation claims            │
│  6. Service B processes request only if attested                  │
│                                                                   │
│  Without valid attestation, token issuance is DENIED.             │
└──────────────────────────────────────────────────────────────────┘
```

### 10.2 Go Code: Attested Service Auth Middleware

```go
// Package attestedauth provides middleware that requires hardware
// attestation for service-to-service communication.
//
// This middleware verifies that incoming requests originate from
// a service running inside a verified TEE enclave. Requests from
// non-attested services are rejected.
package attestedauth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AttestationClaims are JWT claims embedded in attestation tokens.
type AttestationClaims struct {
	jwt.RegisteredClaims
	ServiceName      string `json:"svc"`
	EnclaveMeasurement string `json:"mrenclave"`
	EnclaveSigner     string `json:"mrsigner"`
	TCBLevel          uint16 `json:"tcb"`
	TEEType           string `json:"tee"` // "sgx", "sev-snp", "nitro"
}

// AttestationMiddleware verifies attestation tokens on every request.
type AttestationMiddleware struct {
	verifier      *AttestationVerifier
	aaPublicKey   any // Attestation Authority's signing key
	tokenTTL      time.Duration
	trustedEnclaves map[string]bool // service name → trusted
	mu            sync.RWMutex
}

// NewAttestationMiddleware creates middleware that verifies
// attestation-based service tokens.
func NewAttestationMiddleware(verifier *AttestationVerifier, aaPubKey any) *AttestationMiddleware {
	return &AttestationMiddleware{
		verifier:        verifier,
		aaPublicKey:     aaPubKey,
		tokenTTL:        5 * time.Minute,
		trustedEnclaves: make(map[string]bool),
	}
}

// TrustEnclave adds a service+enclave combination to the trusted list.
func (m *AttestationMiddleware) TrustEnclave(serviceName string, mrenclave [32]byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%x", serviceName, mrenclave)
	m.trustedEnclaves[key] = true
}

// Middleware returns an http.Handler that enforces attestation.
func (m *AttestationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract attestation token from header.
		authHeader := r.Header.Get("X-Attestation-Token")
		if authHeader == "" {
			http.Error(w, `{"error":"attestation_required"}`, http.StatusForbidden)
			return
		}

		// Parse and verify the attestation JWT.
		token, err := jwt.ParseWithClaims(authHeader, &AttestationClaims{}, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return m.aaPublicKey, nil
		})

		if err != nil || !token.Valid {
			slog.Warn("invalid attestation token", "error", err)
			http.Error(w, `{"error":"invalid_attestation"}`, http.StatusForbidden)
			return
		}

		claims, ok := token.Claims.(*AttestationClaims)
		if !ok {
			http.Error(w, `{"error":"invalid_claims"}`, http.StatusForbidden)
			return
		}

		// Check token freshness.
		if time.Since(claims.IssuedAt.Time) > m.tokenTTL {
			http.Error(w, `{"error":"attestation_expired"}`, http.StatusForbidden)
			return
		}

		// Verify the enclave is trusted.
		m.mu.RLock()
		trustKey := fmt.Sprintf("%s:%s", claims.ServiceName, claims.EnclaveMeasurement)
		trusted := m.trustedEnclaves[trustKey]
		m.mu.RUnlock()

		if !trusted {
			slog.Warn("untrusted enclave",
				"service", claims.ServiceName,
				"mrenclave", claims.EnclaveMeasurement,
			)
			http.Error(w, `{"error":"untrusted_enclave"}`, http.StatusForbidden)
			return
		}

		// Add attestation info to context for downstream handlers.
		ctx := context.WithValue(r.Context(), attestationKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type attestationKey struct{}

// GetAttestationFromContext retrieves attestation claims from context.
func GetAttestationFromContext(ctx context.Context) (*AttestationClaims, bool) {
	claims, ok := ctx.Value(attestationKey{}).(*AttestationClaims)
	return claims, ok
}
```

### 10.3 Token Issuance Only to Attested Services

```go
// OAuth service modification: only issue tokens to attested clients.
//
// In the token endpoint, add attestation verification before issuing tokens.

func (s *OAuthService) TokenEndpoint(w http.ResponseWriter, r *http.Request) {
	// Standard OAuth2 validation (client_id, client_secret, grant_type)...

	// NEW: Verify client attestation.
	if s.requireAttestation {
		claims, ok := attestedauth.GetAttestationFromContext(r.Context())
		if !ok {
			writeError(w, "access_denied", "attestation required", http.StatusForbidden)
			return
		}

		// Verify the attested service is authorized for this OAuth client.
		if !s.isServiceAuthorizedForClient(claims.ServiceName, clientID) {
			writeError(w, "access_denied", "service not authorized for client", http.StatusForbidden)
			return
		}

		// Log the attested service identity in audit trail.
		slog.Info("token issued to attested service",
			"client_id", clientID,
			"service", claims.ServiceName,
			"enclave", claims.EnclaveMeasurement,
			"tee_type", claims.TEEType,
		)
	}

	// Proceed with normal token issuance...
}
```

### 10.4 Zero-Trust with Hardware Root of Trust

```
┌───────────────────────────────────────────────────────────────┐
│         Zero-Trust with Hardware Root of Trust                 │
│                                                                │
│  Layer 1: Network (mTLS)                                      │
│  ┌──────────────────────────────────────────────────────┐     │
│  │  Service-to-service mTLS with SPIFFE/SPIRE           │     │
│  │  Certificates issued by cluster CA                    │     │
│  │  Prevents: network MITM, impersonation               │     │
│  │  Does NOT prevent: compromised service identity      │     │
│  └──────────────────────────────────────────────────────┘     │
│                                                                │
│  Layer 2: Software (JWT/OAuth)                                │
│  ┌──────────────────────────────────────────────────────┐     │
│  │  OAuth2 access tokens (GGID OAuth service)            │     │
│  │  Scopes and permissions                               │     │
│  │  Prevents: unauthorized access to resources           │     │
│  │  Does NOT prevent: stolen service credentials         │     │
│  └──────────────────────────────────────────────────────┘     │
│                                                                │
│  Layer 3: Hardware (TEE Attestation)  ← NEW                   │
│  ┌──────────────────────────────────────────────────────┐     │
│  │  Hardware-attested enclave identity                   │     │
│  │  Cryptographic proof of code running in TEE           │     │
│  │  Prevents: stolen credentials, malicious clones,      │     │
│  │            compromised hypervisor, rogue insider      │     │
│  │  Rooted in: CPU hardware + Intel/AMD certificate CA   │     │
│  └──────────────────────────────────────────────────────┘     │
│                                                                │
│  Combined: An attacker must simultaneously compromise          │
│  mTLS certificates + OAuth tokens + TEE hardware to forge      │
│  a request. Hardware attestation makes the last step           │
│  computationally infeasible.                                  │
└───────────────────────────────────────────────────────────────┘
```

---

## 11. GGID Confidential Computing Roadmap

### 11.1 Current State Assessment

Reviewing GGID's current key management codebase:

**File: `pkg/crypto/crypto.go`**
- Password hashing: Argon2id with 64 MB memory cost
- Password pepper: HMAC-SHA256 pre-hash step, set from env var
- AES-256-GCM: symmetric encryption for secrets
- **Gap:** All crypto happens in plaintext process memory

**File: `services/oauth/internal/service/key_rotation.go`**
- `RotatingKeyProvider`: 24h rotation, 24h grace period
- `RotateKey()`: generates new RSA key using `crypto/rand` in process memory
- Key ID: SHA-256 hash of public key (deterministic, stable)
- **Gap:** RSA private key is in process memory; no enclave protection

**File: `services/oauth/internal/server/server.go`**
- `loadOrCreateKeyProvider()`: loads RSA key from PEM file or generates new
- Key files: `configs/rsa_private.pem`, `configs/rsa_public.pem`
- **Gap:** Private key is stored on disk as PEM (plaintext)

**File: `services/auth/internal/service/auth_service.go`**
- JWT signing with `jwt.SigningMethodRS256`
- **Gap:** `kp.PrivateKey()` returns the raw key for JWT library to use

**File: `services/audit/internal/domain/hash_chain.go`**
- HMAC-SHA256 chain for tamper detection
- Secret set from config via `SetHashChainSecret()`
- **Gap:** Hash chain secret in process memory

### 11.2 Migration Path

```
┌──────────────────────────────────────────────────────────────────┐
│                GGID Key Management Maturity Roadmap               │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  STAGE 0: Current State (COMPLETED)                       │   │
│  │                                                           │   │
│  │  • RSA keys: PEM files on disk                            │   │
│  │  • Pepper: environment variable                           │   │
│  │  • RotatingKeyProvider: 24h rotation in memory            │   │
│  │  • All secrets in plaintext process memory                │   │
│  │                                                           │   │
│  │  Threat: Cloud admin, hypervisor, compromised OS can      │   │
│  │          extract all keys                                 │   │
│  └───────────────────────┬──────────────────────────────────┘   │
│                          │                                        │
│  ┌───────────────────────▼──────────────────────────────────┐   │
│  │  STAGE 1: Cloud KMS Integration (RECOMMENDED NEXT)        │   │
│  │                                                           │   │
│  │  • Replace PEM file loading with KMS-backed key storage   │   │
│  │  • Implement CryptoProvider interface (see hsm-kms doc)   │   │
│  │  • Use AWS KMS / GCP KMS / Azure Key Vault for signing    │   │
│  │  • Keys never leave KMS — signing happens in KMS         │   │
│  │                                                           │   │
│  │  Threat mitigated: Disk-based key extraction              │   │
│  │  Remaining: KMS access key compromise, process memory     │   │
│  │  Effort: 2-3 sprints                                     │   │
│  └───────────────────────┬──────────────────────────────────┘   │
│                          │                                        │
│  ┌───────────────────────▼──────────────────────────────────┐   │
│  │  STAGE 2: Confidential VM Deployment                      │   │
│  │                                                           │   │
│  │  • Deploy all GGID services on SEV-SNP Confidential VMs   │   │
│  │  • Zero code changes — entire VM memory encrypted          │   │
│  │  • Add attestation verification for compliance evidence    │   │
│  │  • Encrypt database connections end-to-end                │   │
│  │                                                           │   │
│  │  Threat mitigated: Cloud admin memory access, hypervisor   │   │
│  │  Remaining: TEE side-channel attacks                      │   │
│  │  Effort: 1 sprint (infrastructure only)                   │   │
│  └───────────────────────┬──────────────────────────────────┘   │
│                          │                                        │
│  ┌───────────────────────▼──────────────────────────────────┐   │
│  │  STAGE 3: Enclave-Based Key Management (SGX)              │   │
│  │                                                           │   │
│  │  • Migrate JWT signing to SGX enclave via EGo             │   │
│  │  • Key generation and signing inside enclave              │   │
│  │  • Extend KeyProvider interface with Sign() method        │   │
│  │  • Sealed key persistence across restarts                 │   │
│  │  • Remote attestation for key release                     │   │
│  │                                                           │   │
│  │  Threat mitigated: Process memory extraction               │   │
│  │  Remaining: CPU-level side channels                       │   │
│  │  Effort: 4-5 sprints (significant refactoring)            │   │
│  └───────────────────────┬──────────────────────────────────┘   │
│                          │                                        │
│  ┌───────────────────────▼──────────────────────────────────┐   │
│  │  STAGE 4: Full Confidential Computing (Defense in Depth)  │   │
│  │                                                           │   │
│  │  • HSM for root CA and compliance operations              │   │
│  │  • SGX enclave for JWT signing and PII processing         │   │
│  │  • SEV-SNP CVMs for all microservices                     │   │
│  │  • Attested service mesh                                  │   │
│  │  • Attestation-based token issuance                       │   │
│  │  • Continuous attestation monitoring                      │   │
│  │                                                           │   │
│  │  Threat mitigated: All known software-layer attacks       │   │
│  │  Remaining: Physical hardware attacks, 0-day CPU vulns    │   │
│  │  Effort: Ongoing (compliance-driven)                      │   │
│  └──────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

### 11.3 Design: Enclave-Aware KeyProvider

The following shows how GGID's existing `KeyProvider` interface can be
extended to support enclave-based signing without breaking the current
`RotatingKeyProvider`:

```go
// File: services/oauth/internal/domain/models.go (proposed extension)

// SigningKeyProvider is an extension of KeyProvider that supports
// enclave-based signing where the private key is not available.
type SigningKeyProvider interface {
	KeyProvider // Embed existing interface

	// Sign signs the given data with the current key.
	// For non-enclave providers, this calls PrivateKey().Sign().
	// For enclave providers, this delegates to the TEE.
	Sign(data []byte) ([]byte, error)

	// SignAlgorithm returns the signing algorithm.
	SignAlgorithm() string // "RS256", "ES256", etc.
}

// KeyProvider remains backward-compatible:
// type KeyProvider interface {
//     PublicKey() *rsa.PublicKey
//     PrivateKey() *rsa.PrivateKey  // Returns nil for enclave providers
//     KeyID() string
// }
```

The OAuth service's token generation would be modified:

```go
// Current (services/oauth/internal/service/oauth_service.go):
//
// func (s *OAuthService) issueAccessToken(claims jwt.Claims) (string, error) {
//     token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
//     token.Header["kid"] = s.keyProvider.KeyID()
//     return token.SignedString(s.keyProvider.PrivateKey())
// }

// Proposed (enclave-aware):
//
// func (s *OAuthService) issueAccessToken(claims jwt.Claims) (string, error) {
//     if signingKP, ok := s.keyProvider.(SigningKeyProvider); ok {
//         // Enclave path: sign inside TEE.
//         header := map[string]any{
//             "alg": signingKP.SignAlgorithm(),
//             "typ": "JWT",
//             "kid": signingKP.KeyID(),
//         }
//         signingInput, err := buildSigningInput(header, claims)
//         if err != nil {
//             return "", err
//         }
//         sig, err := signingKP.Sign([]byte(signingInput))
//         if err != nil {
//             return "", fmt.Errorf("enclave signing failed: %w", err)
//         }
//         return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
//     }
//
//     // Fallback: traditional JWT signing with private key.
//     token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
//     token.Header["kid"] = s.keyProvider.KeyID()
//     return token.SignedString(s.keyProvider.PrivateKey())
// }
```

### 11.4 When Confidential Computing Adds Value for GGID

Confidential computing provides measurable value for GGID in these scenarios:

1. **SaaS / Hosted Deployment** — When GGID runs on cloud infrastructure
   that the GGID team does not physically control, CVMs protect tenant
   data from cloud operator access.

2. **Regulated Industries** — Healthcare (HIPAA), Finance (PCI-DSS),
   Government (FedRAMP) increasingly require confidential computing
   for data-in-use protection.

3. **Multi-Tenant SaaS** — Tenants demand proof that their data is
   isolated from other tenants at the hardware level.

4. **High-Value Targets** — IAM systems are prime targets. The signing
   key compromise enables universal impersonation. Enclave-based signing
   raises the bar significantly.

5. **Compliance Evidence** — Attestation reports provide cryptographic
   evidence for auditors that key management operates in hardware-
   protected memory.

**When NOT to invest (yet):**
- Self-hosted single-tenant deployment on controlled hardware
- Development/staging environments
- Low-risk applications where standard TLS + disk encryption suffices

---

## 12. Gap Analysis & Recommendations

### 12.1 Current Gaps

| # | Gap | Risk | Current Mitigation |
|---|---|---|---|
| G1 | RSA private key stored as PEM on disk | Key extraction via disk access | File permissions (0600) |
| G2 | Private key in plaintext process memory | Memory dump extraction | None |
| G3 | Password pepper in process memory | Memory dump extraction | None |
| G4 | No attestation of running services | No proof of execution environment | None |
| G5 | No protection for PII in memory | Memory dump reveals all user data | PostgreSQL RLS (disk only) |
| G6 | No key release based on attestation | Keys could be loaded on compromised hosts | None |
| G7 | No hardware root of trust | Software-only trust chain | mTLS between services |

### 12.2 Action Items

#### Action 1: Deploy GGID on Confidential VMs (Priority: HIGH)

**Effort:** 1 sprint (infrastructure only)
**Risk Reduction:** Protects all PII and keys from cloud operator access

Deploy GGID's microservices on SEV-SNP Confidential VMs in the target
cloud provider:

- **Azure:** DCasv5/ECasv5 series (AMD SEV-SNP)
- **AWS:** `i3en.large` with Nitro Enclaves for signing component
- **GCP:** `n2d-standard-*` with Confidential Computing

No code changes required — only infrastructure (Terraform/Kubernetes manifests).
The entire VM memory is encrypted, protecting all secrets, PII, and tokens.

**Acceptance criteria:**
- All 7 GGID services run on CVM instances
- Attestation report available for compliance evidence
- Service performance within 5% of non-CVM baseline

#### Action 2: Abstract JWT Signing Behind Sign() Interface (Priority: MEDIUM)

**Effort:** 1-2 sprints
**Risk Reduction:** Enables future enclave-based signing without breaking changes

Extend the `domain.KeyProvider` interface with a `Sign()` method and
`SigningKeyProvider` interface. Modify the OAuth service's token
generation to use `Sign()` instead of accessing `PrivateKey()` directly.

This is a prerequisite for Stage 3 (enclave-based signing) and does
not change current behavior — the `RotatingKeyProvider` implements
`Sign()` by delegating to `PrivateKey().Sign()`.

**Acceptance criteria:**
- `SigningKeyProvider` interface defined in `domain/models.go`
- `RotatingKeyProvider` implements `SigningKeyProvider`
- OAuth token generation uses `Sign()` instead of `PrivateKey()`
- All existing tests pass
- `make test` passes with 0 failures

#### Action 3: Implement KMS-Based Key Provider (Priority: MEDIUM)

**Effort:** 2-3 sprints
**Risk Reduction:** Eliminates PEM key files; keys never leave KMS

Implement a `KMSKeyProvider` that implements `SigningKeyProvider` by
delegating signing operations to AWS KMS / Google Cloud KMS / Azure
Key Vault. This replaces the current PEM-file-based key loading.

Reference: `hsm-kms-integration.md` for the `CryptoProvider` interface
design and PKCS#11 integration patterns.

**Acceptance criteria:**
- `KMSKeyProvider` implements `SigningKeyProvider`
- Supports AWS KMS, GCP KMS, Azure Key Vault
- Configurable via environment variables
- Fallback to local PEM for development mode
- Integration tests with LocalStack (AWS KMS emulator)

#### Action 4: Prototype SGX Enclave for JWT Signing (Priority: LOW)

**Effort:** 4-5 sprints (research + development)
**Risk Reduction:** Eliminates process-memory key extraction

Build a proof-of-concept using EGo (edgeless/go) that runs GGID's JWT
signing inside an SGX enclave:

1. Create an enclave binary that:
   - Generates RSA key inside enclave
   - Seals key for persistence
   - Exposes signing via HTTP/gRPC
   - Provides attestation endpoint

2. Modify OAuth service to delegate signing to the enclave

3. Implement attestation verification before key release

4. Benchmark performance (expect <10% overhead with adequate EPC)

**Acceptance criteria:**
- Enclave binary runs on Azure DCsv3 instances
- JWT signing works end-to-end through enclave
- Attestation report verifiable by remote client
- Performance within 10% of native
- Key sealed and restored across enclave restarts

#### Action 5: Attested Service Mesh Integration (Priority: LOW)

**Effort:** 3-4 sprints
**Risk Reduction:** Hardware-rooted service identity

Integrate attestation verification into GGID's service mesh:

1. Deploy an Attestation Authority (AA) service
2. Each service generates attestation quote at startup
3. AA verifies quotes and issues attestation JWTs (5 min TTL)
4. Services present attestation JWT for inter-service calls
5. OAuth service requires attestation for token issuance

**Acceptance criteria:**
- Attestation Authority running in cluster
- All GGID services present valid attestation
- Non-attested services cannot obtain tokens
- Attestation refresh works automatically

### 12.3 Effort Summary

| Action | Priority | Effort | Dependencies |
|---|---|---|---|
| Deploy on CVMs | HIGH | 1 sprint | Cloud account with CVM support |
| Sign() interface | MEDIUM | 1-2 sprints | None |
| KMS Key Provider | MEDIUM | 2-3 sprints | Sign() interface |
| SGX Enclave Prototype | LOW | 4-5 sprints | Sign() interface, EGo |
| Attested Service Mesh | LOW | 3-4 sprints | SGX or SEV-SNP deployment |

**Total estimated effort:** 11-15 sprints for full implementation (Stages 1-4).

**Recommended sequence:**
1. CVM deployment (immediate protection, no code changes)
2. Sign() interface (enables future work)
3. KMS Key Provider (removes PEM files)
4. SGX Enclave Prototype (defense in depth)
5. Attested Service Mesh (zero-trust hardware root)

---

## 13. References

### 13.1 Standards and Specifications

- **Intel SGX SDK Documentation** — https://download.01.org/intel-sgx/latest/linux-latest/docs/
- **Intel SGX DCAP** — https://github.com/intel/SGXDataCenterAttestationPrimitives
- **AMD SEV-SNP Whitepaper** — "SEV-SNP: Strengthening VM Isolation with Integrity Protection and More" (AMD, 2020)
- **Linux SGX Driver** — https://github.com/intel/linux-sgx-driver
- **Confidential Computing Consortium** — https://confidentialcomputing.io/
- **NIST IR 8443** — "Platform Firmware Resiliency Guidelines" (relevant to TEE security)
- **VMware Confidential Computing Initiative** — https://github.com/confidential-containers

### 13.2 Go TEE Frameworks

- **EGo (edgeless/go)** — https://github.com/edgelesssys/ego
- **Gramine** — https://gramineproject.io/
- **Occlum** — https://occlum.io/
- **Confidential Containers** — https://github.com/confidential-containers
- **Marblerun** (SGX orchestration) — https://github.com/edgelesssys/marblerun

### 13.3 Academic Papers

- **Costan et al.** — "Secure Processors Part I: Background, Analysis, and Lessons from SGX" (2018)
- **Foreshadow** — "Foreshadow: Extracting the Keys to the Intel SGX Kingdom with Transient Out-of-Order Execution" (USENIX Security 2018)
- **LVI** — "LVI: Hijacking Transient Execution through Microarchitectural Load Value Injection" (IEEE S&P 2020)
- **SGAxe** — "SGAxe: How SGX Fails in Practice" (2020)
- **CrossLine** — "CrossLine: Breaking 'SGX' Enclaves by Malicious Design" (2021)

### 13.4 Cloud Provider Documentation

- **Azure Confidential Computing** — https://azure.microsoft.com/en-us/solutions/confidential-compute
- **AWS Nitro Enclaves** — https://aws.amazon.com/ec2/nitro/nitro-enclaves/
- **Google Cloud Confidential VMs** — https://cloud.google.com/confidential-computing
- **IBM Cloud Secure Execution** — https://www.ibm.com/cloud/learn/confidential-computing

### 13.5 Related GGID Research Documents

- `hsm-kms-integration.md` — HSM and Cloud KMS integration patterns
- `key-rotation-iam.md` — Key lifecycle and `RotatingKeyProvider` design
- `secret-management-iam.md` — Vault and secret storage patterns
- `zero-trust-iam.md` — Zero-trust architecture for IAM
- `service-mesh-iam.md` — Service mesh integration patterns
- `multi-tenant-isolation.md` — Multi-tenant data isolation
- `post-quantum-cryptography-iam.md` — Post-quantum readiness

---

## Appendix A: Threat Model Summary

| Threat | Current | + CVM | + SGX Enclave | + HSM |
|---|---|---|---|---|
| Disk key extraction | Vulnerable | Protected | Protected | Protected |
| Process memory dump | Vulnerable | **Protected** | **Protected** | Vulnerable (key in KMS proxy) |
| Cloud admin access | Vulnerable | **Protected** | **Protected** | Protected |
| Hypervisor compromise | Vulnerable | **Protected** | **Protected** | Protected |
| Physical DRAM inspection | Vulnerable | **Protected** | **Protected** | Protected |
| CPU side-channel attack | Vulnerable | Vulnerable | Partially mitigated | Protected |
| Supply chain attack | Vulnerable | Partially mitigated | **Protected** (attestation) | Protected |
| Cold boot attack | Vulnerable | **Protected** | **Protected** | Protected |
| Rogue insider (cloud staff) | Vulnerable | **Protected** | **Protected** | Protected |
| FIPS compliance | Not certified | Not certified | Not certified | **Certified** |

## Appendix B: Cost Comparison

| Approach | Setup Cost | Monthly Cost (per node) | Key Benefit |
|---|---|---|---|
| Current (PEM + env vars) | $0 | $0 | Simple, no dependencies |
| Cloud KMS | $0 | $1-3 per million operations | Keys never leave KMS |
| On-prem HSM | $5K-$50K | $200-500 (maintenance) | FIPS certified, physical tamper |
| SEV-SNP CVM | $0 | ~25% cloud surcharge | Full VM memory encryption |
| SGX enclave (EGo) | $0 | ~40% cloud surcharge | Application-level protection |
| Defense in depth (all) | $5K-$50K | ~50% cloud surcharge + KMS fees | Maximum protection |

---

*This document is part of the GGID research series. For implementation
guidance on any of the recommended actions, refer to the companion
documents listed in Section 13.5.*
