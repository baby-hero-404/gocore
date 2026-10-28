# Implementation Map: net/http adapters

**Goal:** Ship `httpmw` and `iam/iamhttp`, share security logic with the Gin adapter, and prove Chi compatibility.
**Tech Stack:** Go `net/http`, `stdres`, `stdlog`, `iam`; Chi only in a throwaway verification module.

---

## Phase 1: httpmw

### Request ID, recovery, logging, CORS

**Why:** One framework-neutral implementation of the rules Gin services already follow.
**Depends on:** none
**Files:** `httpmw/*`, `internal/archtest/archtest_test.go`

**Changes:**
- [x] `ValidRequestID`, `NewRequestID`, `RequestID()`
- [x] `Recovery()` using `stderr.FromPanic` and `stdres.WriteError`
- [x] `RequestLogger()` with a writer that forwards Flush/Hijack/Unwrap, and shared `LogRequest`
- [x] `CORSConfig`, `NewCORSPolicy`, `CORS()`
- [x] archtest rank for `httpmw`

**Verify:**
- [x] Table tests for each scenario in specs.md; WebSocket-style Hijack and Flush pass through the logger

### Gin adapter reuses httpmw

**Depends on:** httpmw
**Files:** `httpserver/middleware.go`

**Changes:**
- [x] `validRequestID` -> `httpmw.ValidRequestID`; CORS decision -> `httpmw.CORSPolicy`; request log -> `httpmw.LogRequest`

**Verify:**
- [x] Existing `httpserver` tests unchanged and green

## Phase 2: iam/iamhttp

**Files:** `iam/iamhttp/*`

**Changes:**
- [x] `JWT(validator, opts...)`, `WithCookie`
- [x] `Casbin(enforcer, resolver...)`, `DefaultResolver`
- [x] `IdentityFromRequest`

**Verify:**
- [x] Tests for every success/failure scenario in specs.md

## Phase 3: Chi compatibility and docs

**Changes:**
- [x] Throwaway module with real Chi: `r.Use(...)`, `r.Group`, `chi.URLParam`, WebSocket-like Hijack
- [x] README for both packages, CHANGELOG (Unreleased), PROFILE/DOMAIN, README package index

**Verify:**
- [x] `make verify` green
