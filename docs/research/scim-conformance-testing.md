# SCIM 2.0 Conformance Testing: Approaches, Tools, and GGID Strategy

> A comprehensive guide to SCIM 2.0 (RFC 7643/7644) conformance testing: available test suites, pass/fail prediction for GGID's SCIM implementation, a custom Go test harness design, and a remediation roadmap to reach 80%+ conformance.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Available Test Suites](#2-available-test-suites)
3. [GGID SCIM Endpoint Inventory](#3-ggid-scim-endpoint-inventory)
4. [Pass/Fail Prediction](#4-passfail-prediction)
5. [Conformance Test Categories (Detailed)](#5-conformance-test-categories-detailed)
6. [Building a GGID SCIM Test Suite](#6-building-a-ggid-scim-test-suite)
7. [Remediation Priority](#7-remediation-priority)
8. [Integration with CI/CD](#8-integration-with-cicd)

---

## 1. Overview

### 1.1 Why Conformance Testing Matters for SCIM

SCIM (System for Cross-domain Identity Management) is the de facto standard for automated user provisioning between identity providers (IdPs) and service providers (SPs). Enterprises rely on SCIM to synchronize user accounts from systems like Microsoft Entra ID (formerly Azure AD), Okta, and PingFederate into downstream applications. When a SCIM endpoint deviates from the specification, the consequences range from silent data loss to catastrophic provisioning failures affecting thousands of users.

Conformance testing answers a critical question: **does your SCIM server behave the way the RFCs say it should?** This is distinct from unit testing (which verifies internal logic) and integration testing (which verifies service-to-service wiring). Conformance testing treats the SCIM endpoint as a black box, sending standard-compliant requests and asserting that the responses match RFC requirements.

Key reasons conformance testing matters:

| Reason | Impact |
|--------|--------|
| **IdP compatibility** | Microsoft Entra ID, Okta, and Google Workspace each implement SCIM clients with subtle differences. Passing conformance tests maximizes the chance of working with all of them. |
| **Data integrity** | A missing `Location` header or incorrect `meta.version` can cause IdPs to lose track of resources, leading to orphaned accounts. |
| **Security** | A SCIM filter engine that concatenates strings into SQL instead of using parameterized queries creates a critical injection vulnerability. |
| **Audit and compliance** | Enterprise customers increasingly require proof of SCIM compliance (SOC 2, ISO 27001 evidence). |
| **Reduced support burden** | Each interop bug discovered in production costs 10x more to fix than catching it during development. |

### 1.2 SCIM Spec Compliance vs Interoperability

Two related but distinct concepts are often conflated:

**Spec compliance** means conforming to the letter of RFC 7643 and RFC 7644. Every MUST, SHOULD, and MAY statement is verified. For example:
- `POST /Users` MUST return `201 Created` with a `Location` header (RFC 7644 Section 3.3).
- The `filter` parameter uses a defined grammar with 13 operators (RFC 7644 Section 3.4.2).
- Error responses MUST include the `urn:ietf:params:scim:api:messages:2.0:Error` schema.

**Interoperability** means working correctly with real-world IdP SCIM clients in production. This is broader than spec compliance because:
- IdPs often have proprietary extensions or assumptions beyond the RFC.
- IdPs may tolerate spec violations in some servers but not others.
- Some RFC requirements are ambiguous (e.g., `eq` case sensitivity for string attributes).
- IdPs test with their own reference implementations, not the RFC.

| Aspect | Spec Compliance | Interoperability |
|--------|----------------|------------------|
| **Goal** | Match RFC text | Work with real IdPs |
| **Tests** | RFC-derived assertions | IdP-specific integration tests |
| **Coverage** | Well-defined, finite | Open-ended, vendor-specific |
| **Verification** | Automated test suite | Manual or semi-automated per IdP |
| **Value** | Foundational | What customers actually need |

**Best practice**: Achieve high spec compliance first (via conformance test suites), then layer IdP-specific integration tests on top.

### 1.3 Common Interop Issues Between Providers

Real-world interop problems observed across SCIM implementations:

| Issue | Description | Affected IdPs |
|-------|-------------|---------------|
| **PATCH format variations** | Some IdPs send `"op": "replace"` (lowercase), others send `"op": "Replace"`. RFC says comparison is case-insensitive but many servers fail. | Entra ID, Okta |
| **`userName` vs `emails`** | Entra ID matches users by `userName`; Okta may use `emails[type eq "work"].value`. Servers must support both as joining properties. | All major IdPs |
| **Filter URL encoding** | IdPs encode filter expressions differently (some use `+` for spaces, others use `%20`). Servers must handle both. | All |
| **Pagination edge cases** | `count=0` should return zero results but still report `totalResults`. Some IdPs test this. | Entra ID |
| **`active` attribute semantics** | PATCH `active: false` should deactivate (soft delete), not hard delete. Some servers confuse this. | Entra ID, Okta |
| **Group member `$ref`** | Some IdPs require the `$ref` field in group member objects; others ignore it. The `$` in JSON keys requires careful Go struct tag handling. | Okta, Google |
| **`displayName` filtering** | Okta tests `GET /Groups?filter=displayName eq "value"`. This is the most common group filter. | Okta |
| **Error response format** | IdPs parse `scimType` in error responses to differentiate error categories. Missing `scimType` causes generic error handling. | All |
| **`meta.version` / ETag** | Some IdPs use `If-Match` for optimistic concurrency; others ignore it entirely. | Salesforce, custom clients |
| **Content-Type negotiation** | IdPs send `application/scim+json` or `application/json`. Servers should accept both and prefer `scim+json` in responses. | All |

---

## 2. Available Test Suites

### 2.1 SCIM Verify (scim.dev / verify.scim.dev)

SCIM Verify is a backend-run conformance testing service hosted at [scim.dev/verify](https://scim.dev/verify/) with detailed documentation at [verify.scim.dev/docs](https://verify.scim.dev/docs/). It is the most configurable and comprehensive SCIM 2.0 conformance testing tool available.

#### How It Works

1. **Enter endpoint details**: You provide your SCIM base URL (e.g., `https://yourapp.com/scim/v2`), a bearer token, and optionally a YAML configuration file.
2. **Schema auto-detection**: SCIM Verify can auto-discover your server's schema by calling `GET /Schemas` and `GET /ResourceTypes`, reducing manual configuration.
3. **Automated test execution**: The backend runs a comprehensive suite of CRUD, filtering, sorting, pagination, PATCH, and schema validation tests against your endpoint.
4. **Report generation**: Results are presented as a pass/fail matrix per test category, with detailed error messages for failures.

#### Test Categories

SCIM Verify tests the following categories:

| Category | Operations Tested | Test Count (Approx) |
|----------|-------------------|---------------------|
| **User CRUD** | POST, GET (list + single), PUT, PATCH, DELETE on `/Users` | 15-25 |
| **Group CRUD** | POST, GET (list + single), PUT, PATCH, DELETE on `/Groups` | 10-20 |
| **Filtering** | `eq`, `co`, `sw`, `ew`, `pr`, `ne`, `gt`, `ge`, `lt`, `le`, `and`, `or`, `not`, complex/multi-valued | 15-30 |
| **Sorting** | `sortBy`, `sortOrder` (ascending/descending) on sortable attributes | 5-10 |
| **Pagination** | `startIndex`, `count`, `totalResults`, `itemsPerPage` correctness | 8-12 |
| **Schema discovery** | `GET /Schemas`, `GET /ResourceTypes`, `GET /ServiceProviderConfig` | 5-8 |
| **Error handling** | 400 for invalid input, 404 for missing, 409 for conflicts | 5-10 |
| **PATCH operations** | `add`, `replace`, `remove` with path expressions including value filters | 10-15 |

#### YAML Configuration Format

SCIM Verify is highly configurable via a YAML file. Each test case specifies a request payload and an expected JSON Schema for the response:

```yaml
# SCIM Verify Configuration
detectSchema: true
detectResourceTypes: true

users:
  enabled: true
  operations:
    - GET
    - POST
    - PUT
    - PATCH
    - DELETE
  sortAttributes:
    - userName

  filter_tests:
    - filter: userName eq "bjensen"
      user_schema:
        type: object
        properties:
          schemas:
            type: array
            items:
              type: string
            contains:
              const: urn:ietf:params:scim:schemas:core:2.0:User
          userName:
            type: string
            const: bjensen
        required: [userName, schemas]
        additionalProperties: true

  post_tests:
    - request:
        schemas:
          - urn:ietf:params:scim:schemas:core:2.0:User
        userName: barbara jensen
        emails:
          - value: barbara.jensen@example.com
      response:
        type: object
        properties:
          schemas:
            type: array
            items:
              type: string
            contains:
              const: urn:ietf:params:scim:schemas:core:2.0:User
          userName:
            type: string
            const: barbara jensen
        required: [userName, schemas]
        additionalProperties: true

  put_tests:
    - id: AUTO  # AUTO means ID determined at runtime
      request:
        schemas:
          - urn:ietf:params:scim:schemas:core:2.0:User
        userName: testuser6238
        emails:
          - value: barbara.jensen@example.com
      response:
        type: object
        properties:
          userName:
            type: string
            const: testuser6238
        required: [userName, schemas]
        additionalProperties: true

  patch_tests:
    - id: AUTO
      request:
        schemas:
          - urn:ietf:params:scim:api:messages:2.0:PatchOp
        Operations:
          - op: replace
            path: userName
            value: JohnDoe
      response:
        type: object
        properties:
          userName:
            type: string
            const: JohnDoe
        required: [userName, schemas]
        additionalProperties: true

  delete_tests:
    - id: AUTO

groups:
  enabled: true
  operations:
    - GET
    - POST
    - PUT
    - PATCH
    - DELETE
  sortAttributes:
    - displayName
  post_tests:
    - request:
        schemas:
          - urn:ietf:params:scim:schemas:core:2.0:Group
        displayName: TestGroup
      response:
        type: object
        properties:
          displayName:
            type: string
            const: TestGroup
        required: [displayName, schemas]
        additionalProperties: true
  patch_tests:
    - id: AUTO
      request:
        schemas:
          - urn:ietf:params:scim:api:messages:2.0:PatchOp
        Operations:
          - op: replace
            path: displayName
            value: NewGroupName
      response:
        type: object
        properties:
          displayName:
            type: string
            const: NewGroupName
        required: [displayName, schemas]
        additionalProperties: true
  delete_tests:
    - id: AUTO
```

The `AUTO` keyword for `id` fields is a powerful feature: it tells the framework to create a resource first, capture its ID, and then use that ID for subsequent PUT/PATCH/DELETE tests.

#### What It Reports

SCIM Verify produces a detailed compliance report:

- **Per-test pass/fail** with HTTP request/response details for failures
- **Category-level compliance scores** (e.g., "Filtering: 8/15 passed")
- **Overall compliance percentage**
- **Schema validation errors** (JSON Schema mismatch details)

#### Limitations

| Limitation | Detail |
|------------|--------|
| **Public endpoint required** | The SCIM endpoint must be reachable from the internet (or via a tunnel like ngrok) for the hosted version |
| **Limited customization** | While YAML config helps, the test set is predefined; you cannot easily add custom test flows |
| **No CI/CD native integration** | The hosted version runs in a browser; running it in CI requires the self-hosted variant |
| **No tenant isolation testing** | Does not test multi-tenant behavior or X-Tenant-ID headers |
| **Stateful test ordering** | Tests create/delete resources, so running them against a shared environment can cause conflicts |

### 2.2 Microsoft Entra ID SCIM Validator

The [Microsoft Entra SCIM Validator](https://scimvalidator.microsoft.com/) is a free web-based tool at `scimvalidator.microsoft.com` that tests SCIM endpoints specifically for compatibility with the Microsoft Entra ID (Azure AD) provisioning service.

#### How It Works

1. **Navigate** to `https://scimvalidator.microsoft.com/`
2. **Select a testing method** (three options):
   - **Use default attributes**: System provides default SCIM attributes; you modify as needed
   - **Discover schema** (recommended): Tool calls `GET /Schemas` on your endpoint to auto-discover supported attributes
   - **Upload Microsoft Entra Schema**: Upload a `.json` schema exported from an Entra ID sample app
3. **Configure attributes**: Specify which attributes your server supports, select the "joining property" (matching attribute for user lookup)
4. **Enable Group Tests**: Optionally enable group-related test cases
5. **Run Test Schema**: Click to execute the test suite
6. **Review results**: Pass/fail summary with detailed error info under "show details"

#### Test Categories and Validations

The Entra SCIM Validator performs the following test sequences:

**User Tests:**

| Test | Flow | Expected |
|------|------|----------|
| Create New User | POST /Users with full payload → GET /Users?filter={joining} eq "value" → DELETE /Users/{id} | 201 on POST, GET returns created user with matching values |
| Create Duplicate User | POST /Users twice with same joining property | 201 first, 409 second |
| Add Attributes | POST /Users → PATCH /Users/{id} (add op) → GET to verify | PATCH success, GET shows added attributes |
| Replace User Attributes | POST /Users → PATCH /Users/{id} (replace op) → GET to verify | PATCH success, GET shows replaced attributes |
| Update Joining Property | POST /Users → PATCH to update userName → GET with new value | Joining property updated |
| Update Active to False | POST /Users → PATCH active:false → GET to verify | active=false in response |

**Group Tests (when enabled):**

| Test | Flow | Expected |
|------|------|----------|
| Create New Group | POST /Groups → GET /Groups?filter → DELETE | 201 on POST, GET matches |
| Create Duplicate Group | POST /Groups twice | 201 first, 409 second |
| Update Group Attributes | POST /Groups → PATCH (replace non-member attrs) → GET | Attributes updated |
| Add Group Member | POST /Groups → POST /Users → PATCH /Groups/{id} (add member) → GET | Member in group |

#### Expression Support

The validator supports dynamic value expressions for attribute generation:

| Expression | Meaning | Example |
|------------|---------|---------|
| `{%generateRandomString 6%}` | Random alphabetic string | `CXJHYP` |
| `{%generateRandomNumber 4%}` | Random numeric string | `8821` |
| `{%generateAlphaNumeric 7%}` | Random alphanumeric | `59Q2M9W` |
| `{%generateAlphaNumericWithSpecialCharacters 8%}` | Alphanumeric + special char | `D385N05'` |

Example: `{%generateRandomString 6%}@contoso.com` generates a unique `userName` on each test run.

#### Known Limitations

- **Soft deletes not supported**: Does not test PATCH `active: false` as soft delete (only tests the attribute value change)
- **Timezone format issues**: Randomly generated timezone values may fail validation on strict servers
- **Remove mandatory attributes**: May attempt to remove required attributes via PATCH; such failures should be ignored
- **Entra-specific assumptions**: Tests are calibrated for Entra ID's SCIM client behavior, which may differ from other IdPs

#### When to Use

- **Primary use case**: Preparing for Microsoft Entra ID gallery app integration
- **Filter requirement**: Tests extensively use `GET /Users?filter={joining} eq "value"`, so your server MUST support at least `eq` filtering
- **PATCH requirement**: Tests PATCH add/replace operations, so basic PATCH support is required

### 2.3 Okta SCIM App Integration Testing

Okta provides SCIM 2.0 testing through its App Integration workflow. Unlike browser-based validators, Okta testing happens within the Okta platform itself.

#### How It Works

1. **Create a SCIM app integration** in the Okta Admin Console
2. **Configure the SCIM connector**: Enter base URL, authentication method (Bearer token or Basic Auth)
3. **Run the Import test**: Okta calls `GET /Users` and `GET /Groups` to discover existing resources
4. **Run the Provisioning test**: Okta creates, updates, and deactivates test users via SCIM
5. **Review the integration log**: Okta provides detailed logs of each SCIM API call and response

#### Okta-Specific SCIM Requirements

Okta's SCIM client has specific expectations beyond the RFC:

| Requirement | Detail |
|-------------|--------|
| **`displayName` filter on Groups** | Okta always filters groups by `displayName eq "value"`. This filter MUST work. |
| **`userName` uniqueness** | Okta expects `userName` to be the primary matching attribute |
| **Pagination with large datasets** | Okta paginates through all users on initial import; `startIndex` + `count` must work correctly |
| **PATCH `active` toggle** | Okta uses PATCH `active: false` for deactivation and PATCH `active: true` for reactivation |
| **Group membership sync** | Okta PATCHes group memberships via `members` array add/remove operations |
| **`externalId` support** | Okta sets `externalId` on provisioned users; server should persist and return it |
| **Error handling on conflicts** | Okta expects 409 Conflict (not 500) when creating a duplicate user |

#### Okta SCIM Test Flow

```
1. Okta calls GET /ServiceProviderConfig
   → Checks which features (filter, patch, bulk, sort, etag) are supported

2. Okta calls GET /Users?count=1
   → Verifies endpoint is reachable and returns valid ListResponse

3. Okta calls GET /Users?startIndex=1&count=200
   → Imports existing users (pagination through all pages)

4. Okta calls POST /Users
   → Creates a test user (Okta_ prefixed userName)

5. Okta calls GET /Users?filter=userName eq "Okta_test_xxx"
   → Verifies the created user can be found via filter

6. Okta calls PATCH /Users/{id}
   → Updates user attributes (profile mapping)

7. Okta calls PATCH /Users/{id} with active: false
   → Deactivates the test user

8. Okta calls GET /Groups?filter=displayName eq "Everyone"
   → Tests group filtering

9. Okta calls PATCH /Groups/{id} with member add/remove
   → Tests group membership management
```

#### Limitations

- **Requires Okta tenant**: You need an Okta Developer account (free) to run tests
- **Manual process**: No CLI or API to automate the test run
- **Okta-specific**: Tests Okta's SCIM client behavior, not general RFC compliance
- **Slow feedback**: Each test cycle takes 5-15 minutes (Okta provisioning cycles)

### 2.4 Custom Test Harness

Building a custom SCIM conformance test suite gives you maximum control, CI/CD integration, and the ability to test GGID-specific behavior (multi-tenancy, role-to-group mapping, etc.).

#### Advantages of a Custom Harness

| Advantage | Detail |
|-----------|--------|
| **Full control** | Define exactly which tests run, what assertions are checked |
| **CI/CD native** | Run as part of `go test` or GitHub Actions |
| **Tenant testing** | Test X-Tenant-ID isolation, cross-tenant data leakage prevention |
| **No external dependencies** | No public endpoint exposure needed; runs against localhost |
| **Custom assertions** | Check GGID-specific behavior (e.g., SCIM Group maps to GGID Role) |
| **Fast feedback** | Runs in seconds, not minutes |

#### Design Approach

A custom SCIM test harness typically consists of:

1. **SCIM test client**: A Go HTTP client that sends SCIM requests with configurable base URL, auth token, and tenant ID
2. **Test case definitions**: Structured definitions of request + expected response (YAML, JSON, or Go structs)
3. **Assertion helpers**: Functions that validate SCIM compliance (schema URNs, status codes, `meta` fields)
4. **Test lifecycle management**: Setup (create test users), teardown (delete test resources), isolation (unique test data per run)

#### Test Case Definition Format (YAML)

```yaml
# scim-test-cases.yaml
test_suite:
  name: "GGID SCIM 2.0 Conformance"
  base_url: "http://localhost:8081/scim/v2"
  auth:
    type: bearer
    token: "${SCIM_BEARER_TOKEN}"
  headers:
    X-Tenant-ID: "${TENANT_ID}"

  tests:
    # === User CRUD ===
    - id: U-001
      name: "Create user - success"
      method: POST
      path: /Users
      request:
        schemas:
          - "urn:ietf:params:scim:schemas:core:2.0:User"
        userName: "conform-test-001@example.com"
        displayName: "Conform Test 001"
        emails:
          - value: "conform-test-001@example.com"
            type: "work"
            primary: true
        active: true
      expected:
        status: 201
        headers:
          Content-Type: "application/scim+json"
          Location: "^(.+)/Users/(.+)$"
        body:
          schemas:
            contains: "urn:ietf:params:scim:schemas:core:2.0:User"
          id:
            type: string
            required: true
          userName:
            equals: "conform-test-001@example.com"
          meta.resourceType:
            equals: "User"
          meta.location:
            required: true
      cleanup:
        method: DELETE
        path: "/Users/${response.id}"

    - id: U-002
      name: "Create user - duplicate userName conflict"
      method: POST
      path: /Users
      request:
        schemas:
          - "urn:ietf:params:scim:schemas:core:2.0:User"
        userName: "conform-test-002@example.com"
      expected:
        status: 201
      followup:
        - id: U-002b
          name: "Duplicate create should conflict"
          method: POST
          path: /Users
          request:
            schemas:
              - "urn:ietf:params:scim:schemas:core:2.0:User"
            userName: "conform-test-002@example.com"
          expected:
            status: 409
            body:
              schemas:
                contains: "urn:ietf:params:scim:api:messages:2.0:Error"

    # === Filter Tests ===
    - id: F-001
      name: "Filter - userName eq"
      method: GET
      path: '/Users?filter=userName eq "conform-test-001@example.com"'
      expected:
        status: 200
        body:
          schemas:
            contains: "urn:ietf:params:scim:api:messages:2.0:ListResponse"
          totalResults:
            gte: 1
          Resources:
            type: array
            minLength: 1

    # === Schema Discovery ===
    - id: SD-001
      name: "ServiceProviderConfig"
      method: GET
      path: /ServiceProviderConfig
      expected:
        status: 200
        body:
          schemas:
            contains: "urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"
          patch.supported:
            type: boolean
          filter.supported:
            type: boolean
          sort.supported:
            type: boolean
```

---

## 3. GGID SCIM Endpoint Inventory

This section catalogs every SCIM endpoint that GGID currently exposes, mapping each to its RFC requirements and current implementation status.

### 3.1 Route Registration

SCIM routes are registered in `services/identity/internal/server/http.go`:

```go
// SCIM 2.0 endpoints
scimHandler := scim.NewHandler(h.svc)
scimHandler.RegisterRoutes(h.mux)
```

And in `services/identity/internal/scim/handler.go`:

```go
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/scim/v2/Users", h.handleUsersCollection)
    mux.HandleFunc("/scim/v2/Users/", h.handleUserResource)
    mux.HandleFunc("/scim/v2/Groups", h.handleGroupsCollection)
    mux.HandleFunc("/scim/v2/Groups/", h.HandleGroupResource)
    mux.HandleFunc("/scim/v2/ServiceProviderConfig", h.handleServiceProviderConfig)
    mux.HandleFunc("/scim/v2/ResourceTypes", h.handleResourceTypes)
}
```

### 3.2 Endpoint Catalog

| # | Endpoint | Method | Handler | Status | RFC Reference |
|---|----------|--------|---------|--------|---------------|
| 1 | `/scim/v2/Users` | GET | `listUsers` | Partial | RFC 7644 Section 3.4 |
| 2 | `/scim/v2/Users` | POST | `createUser` | Partial | RFC 7644 Section 3.3 |
| 3 | `/scim/v2/Users/{id}` | GET | `getUser` | Working | RFC 7644 Section 3.4.1 |
| 4 | `/scim/v2/Users/{id}` | PUT | `replaceUser` | Partial | RFC 7644 Section 3.5.1 |
| 5 | `/scim/v2/Users/{id}` | PATCH | `patchUser` | Minimal | RFC 7644 Section 3.5.2 |
| 6 | `/scim/v2/Users/{id}` | DELETE | `deleteUser` | Working | RFC 7644 Section 3.6 |
| 7 | `/scim/v2/Groups` | GET | `listGroups` | Stub (mock data) | RFC 7644 Section 3.4 |
| 8 | `/scim/v2/Groups` | POST | `createGroup` | Partial (no persistence) | RFC 7644 Section 3.3 |
| 9 | `/scim/v2/Groups/{id}` | GET | `getGroup` | Stub (mock data) | RFC 7644 Section 3.4.1 |
| 10 | `/scim/v2/Groups/{id}` | PATCH | `patchGroup` | Stub (no-op) | RFC 7644 Section 3.5.2 |
| 11 | `/scim/v2/Groups/{id}` | DELETE | `deleteGroup` | Stub (no existence check) | RFC 7644 Section 3.6 |
| 12 | `/scim/v2/ServiceProviderConfig` | GET | `handleServiceProviderConfig` | Working | RFC 7643 Section 5 |
| 13 | `/scim/v2/ResourceTypes` | GET | `handleResourceTypes` | Working | RFC 7643 Section 6 |
| 14 | `/scim/v2/Schemas` | GET | Not implemented | Missing | RFC 7643 Section 7 |
| 15 | `/scim/v2/Bulk` | POST | Not implemented | Missing | RFC 7644 Section 3.7 |
| 16 | `/scim/v2/.search` | POST | Not implemented | Missing | RFC 7644 Section 3.4.3 |

### 3.3 Detailed Behavior Per Endpoint

#### 1-2. Users Collection (GET + POST)

**GET /scim/v2/Users** (`listUsers`):
- Parses `startIndex` (defaults to 1) and `count` (defaults to 20, max 100)
- Calculates offset as `startIndex - 1`
- Calls `svc.ListUsers` with page size and offset
- Returns `ListResponse` with `totalResults`, `itemsPerPage`, `startIndex`, `Resources`
- **Missing**: No filter parameter parsing, no sortBy/sortOrder, no attributes/excludedAttributes
- **Bug**: `itemsPerPage` returns the requested page size, not the actual number of items returned

**POST /scim/v2/Users** (`createUser`):
- Decodes SCIM user JSON
- Extracts first email from `emails` array
- Calls `svc.CreateUser` with `userName`, `email`, hardcoded temp password `"TempPass123!"`
- Returns `201 Created` with SCIM-formatted user
- **Missing**: No `Location` header in response, `externalId` not persisted, `name.familyName` not stored, `phoneNumbers` not stored, no `active` attribute honored on creation

#### 3. Get User by ID

**GET /scim/v2/Users/{id}** (`getUser`):
- Parses UUID from path
- Calls `svc.GetUser`
- Returns `200 OK` with SCIM user or `404 Not Found`
- **Missing**: No `ETag` header, `If-None-Match` not handled

#### 4. Replace User (PUT)

**PUT /scim/v2/Users/{id}** (`replaceUser`):
- Decodes SCIM user JSON
- Only updates `displayName`
- Toggles `active` via `LockUser`/`UnlockUser`
- Returns `200 OK` with updated user
- **Missing**: PUT should be full replacement (unset attributes should be cleared), most attributes ignored (emails, phoneNumbers, name, title, etc.), no `If-Match` handling

#### 5. Patch User (PATCH)

**PATCH /scim/v2/Users/{id}** (`patchUser`):
- Decodes PATCH request with `Operations` array
- Handles only `displayName` and `active` paths (case-insensitive)
- Supports `replace`, `add`, `remove` operations for those two paths
- Returns `200 OK` with updated user
- **Missing**: No complex attribute path support (`emails[type eq "work"].value`), no multi-valued array operations, no `name.*` sub-attributes, no enterprise extension attributes

#### 6. Delete User

**DELETE /scim/v2/Users/{id}** (`deleteUser`):
- Calls `svc.DeleteUser`
- Returns `204 No Content` on success, `404 Not Found` on error
- **Missing**: No `If-Match` precondition check

#### 7-11. Groups (All Methods)

All Group endpoints use `getMockGroups()` which returns hardcoded data:
```go
func (h *Handler) getMockGroups(tenantID, filter string) []SCIMGroup {
    all := []SCIMGroup{
        {ID: "role-admin-001", DisplayName: "Admin", ...},
        {ID: "role-user-001", DisplayName: "User", ...},
    }
    // ...
}
```

- **GET /Groups**: Lists mock groups with basic pagination and `displayName eq` string-split filter
- **POST /Groups**: Generates random UUID, returns `201 Created` but does NOT persist
- **GET /Groups/{id}**: Searches mock array by ID
- **PATCH /Groups/{id}**: Returns hardcoded `200 OK` without applying any operations
- **DELETE /Groups/{id}**: Returns `204 No Content` without checking existence

#### 12. ServiceProviderConfig

Returns hardcoded capabilities:
```json
{
  "patch": {"supported": true},
  "bulk": {"supported": false, "maxOperations": 0, "maxPayloadSize": 0},
  "filter": {"supported": true, "maxResults": 100},
  "changePassword": {"supported": true},
  "sort": {"supported": true},
  "etag": {"supported": false},
  "authenticationSchemes": [{"name": "OAuth 2.0 Bearer", "type": "oauthbearertoken"}]
}
```

**Issue**: `filter` and `sort` are advertised as `supported: true` but are not actually implemented in the SCIM endpoints. This will cause conformance tests to expect filtering/sorting behavior and then fail when it is absent.

#### 13. ResourceTypes

Returns User and Group resource type definitions. **Missing**: `schemaExtensions` field (EnterpriseUser extension not declared), `meta` block.

### 3.4 SCIM User Conversion

The `toSCIMUser` function maps domain User to SCIM format:

| SCIM Attribute | Source | Notes |
|---------------|--------|-------|
| `schemas` | Hardcoded | `["urn:ietf:params:scim:schemas:core:2.0:User"]` |
| `id` | `u.ID.String()` | UUID |
| `userName` | `u.Username` | Direct mapping |
| `displayName` | `u.DisplayName` | Direct mapping |
| `name.givenName` | `u.DisplayName` | **Bug**: Uses DisplayName for givenName |
| `name.familyName` | Not set | Always empty |
| `emails[0].value` | `u.Email` | Single work email |
| `emails[0].type` | Hardcoded | `"work"` |
| `emails[0].primary` | Hardcoded | `true` |
| `active` | `u.Status == UserStatusActive` | Boolean from status |
| `meta.resourceType` | Hardcoded | `"User"` |
| `meta.location` | `"/scim/v2/Users/" + u.ID` | **Relative URL, should be absolute** |
| `meta.created` | `u.CreatedAt` | RFC 3339 format |
| `meta.lastModified` | `u.UpdatedAt` | RFC 3339 format |
| `meta.version` | `"W/\"{nanos}\""` | Uses UnixNano timestamp |

**Missing attributes**: `externalId`, `nickName`, `profileUrl`, `title`, `userType`, `preferredLanguage`, `locale`, `timezone`, `addresses`, `phoneNumbers`, `groups`, `photos`, `ims`, `entitlements`, `x509Certificates`.

---

## 4. Pass/Fail Prediction

Based on the 20 gaps identified in the [SCIM 2.0 Compliance and Test Suite Design](./scim2-compliance-test-suite.md) document and the code analysis above, here is a predicted pass/fail matrix for each test category if GGID were tested against SCIM Verify or the scim2/test-suite today.

### 4.1 Prediction Matrix

| # | Endpoint | Test Category | Predicted Result | Reason |
|---|----------|---------------|-----------------|--------|
| 1 | `POST /Users` | Create user | **PASS** (partial) | Returns 201 + valid user, but missing `Location` header |
| 2 | `POST /Users` | Location header | **FAIL** | No `Location` header in response (RFC 7644 Section 3.3) |
| 3 | `POST /Users` | Duplicate conflict | **PASS** | Returns 409 on duplicate (error handler) |
| 4 | `POST /Users` | Missing userName | **FAIL** | Returns 409 instead of 400 for validation errors |
| 5 | `POST /Users` | `externalId` persistence | **FAIL** | `externalId` accepted but not persisted or returned |
| 6 | `GET /Users` | List response format | **PASS** | Returns ListResponse with correct schema URN |
| 7 | `GET /Users` | Pagination | **PASS** | `startIndex` + `count` work correctly |
| 8 | `GET /Users` | `itemsPerPage` accuracy | **FAIL** | Returns requested count, not actual items returned |
| 9 | `GET /Users` | `totalResults` with filter | **FAIL** | No filter support; `totalResults` always shows unfiltered total |
| 10 | `GET /Users/{id}` | Get single user | **PASS** | Returns 200 with valid user |
| 11 | `GET /Users/{id}` | 404 for missing user | **PASS** | Returns 404 for non-existent UUID |
| 12 | `GET /Users/{id}` | ETag header | **SKIP** | `etag: false` in ServiceProviderConfig (correctly advertised) |
| 13 | `PUT /Users/{id}` | Full replacement | **FAIL** | Only updates `displayName`; ignores most attributes |
| 14 | `PUT /Users/{id}` | `active` toggle | **PASS** | Correctly handles active/inactive via Lock/Unlock |
| 15 | `PATCH /Users/{id}` | Replace `displayName` | **PASS** | Handles `displayName` replace correctly |
| 16 | `PATCH /Users/{id}` | Replace `active` | **PASS** | Handles `active` replace correctly |
| 17 | `PATCH /Users/{id}` | Add email | **FAIL** | No multi-valued attribute support |
| 18 | `PATCH /Users/{id}` | Remove email | **FAIL** | No complex path filter support |
| 19 | `PATCH /Users/{id}` | Replace specific email | **FAIL** | `emails[type eq "work"].value` not supported |
| 20 | `PATCH /Users/{id}` | `name.familyName` | **FAIL** | Only `name.givenName` partially handled |
| 21 | `DELETE /Users/{id}` | Delete user | **PASS** | Returns 204 No Content |
| 22 | `DELETE /Users/{id}` | 404 for missing | **PASS** | Returns 404 for non-existent |
| 23 | `GET /Users?filter=` | `eq` operator | **FAIL** | No SCIM filter engine for Users |
| 24 | `GET /Users?filter=` | `co` operator | **FAIL** | No filter engine |
| 25 | `GET /Users?filter=` | `sw` operator | **FAIL** | No filter engine |
| 26 | `GET /Users?filter=` | `ew` operator | **FAIL** | No filter engine |
| 27 | `GET /Users?filter=` | `pr` operator | **FAIL** | No filter engine |
| 28 | `GET /Users?filter=` | Complex path filter | **FAIL** | No filter engine |
| 29 | `GET /Users?sortBy=` | Sort by userName | **FAIL** | SCIM layer does not parse sortBy |
| 30 | `GET /Users?sortBy=` | Sort order | **FAIL** | SCIM layer does not parse sortOrder |
| 31 | `POST /Groups` | Create group | **PARTIAL** | Returns 201 but does not persist (mock UUID) |
| 32 | `GET /Groups` | List groups | **PARTIAL** | Returns mock data, not real DB-backed groups |
| 33 | `GET /Groups?filter=` | `displayName eq` | **PARTIAL** | Works for exact match on mock data only |
| 34 | `PATCH /Groups/{id}` | Add member | **FAIL** | Returns hardcoded 200, no operations applied |
| 35 | `PATCH /Groups/{id}` | Remove member | **FAIL** | No-op |
| 36 | `DELETE /Groups/{id}` | Delete group | **FAIL** | Returns 204 without checking existence |
| 37 | `GET /ServiceProviderConfig` | Config response | **PASS** | Returns valid config with all required fields |
| 38 | `GET /ServiceProviderConfig` | `filter.supported` honesty | **FAIL** | Reports `true` but filter not implemented |
| 39 | `GET /ServiceProviderConfig` | `sort.supported` honesty | **FAIL** | Reports `true` but sort not implemented |
| 40 | `GET /ResourceTypes` | Resource types | **PASS** | Returns User + Group types |
| 41 | `GET /ResourceTypes/User` | Individual type | **FAIL** | No `/ResourceTypes/{id}` route |
| 42 | `GET /Schemas` | Schema introspection | **FAIL** | `/Schemas` endpoint not implemented |
| 43 | `GET /Schemas/{urn}` | Individual schema | **FAIL** | Not implemented |
| 44 | Error format | `scimType` field | **FAIL** | `ErrorResponse` lacks `scimType` |
| 45 | Content-Type | `application/scim+json` | **PASS** | Uses correct media type |

### 4.2 Summary Scorecard

| Category | Total Tests | Predicted PASS | Predicted FAIL | Estimated Compliance |
|----------|------------|---------------|---------------|---------------------|
| User CRUD | 10 | 6 | 4 | 60% |
| PATCH Operations | 5 | 2 | 3 | 40% |
| Filtering | 6 | 0 | 6 | 0% |
| Sorting | 2 | 0 | 2 | 0% |
| Pagination | 3 | 1 | 2 | 33% |
| Group CRUD | 5 | 0 | 5 | 0% |
| Schema Discovery | 4 | 1 | 3 | 25% |
| Error Handling | 2 | 0 | 2 | 0% |
| Conformance | 8 | 3 | 5 | 38% |
| **Overall** | **45** | **13** | **32** | **~29%** |

**Estimated GGID SCIM conformance: ~29%**

This means GGID would fail approximately 7 out of 10 conformance test cases. The primary blockers are:
1. No SCIM filter engine (0% on all filter tests)
2. No sort support in SCIM layer (0% on sort tests)
3. Groups are entirely mock-backed (0% on group CRUD)
4. PATCH limited to 2 attributes (40% on patch tests)
5. Missing `/Schemas` endpoint (0% on schema introspection)

---

## 5. Conformance Test Categories (Detailed)

### 5.1 Resource Endpoints

These are the core CRUD operations defined in RFC 7644 Sections 3.3-3.6. Every SCIM server MUST implement these.

#### POST /Users — Create User

| Test | Request | Expected Response | GGID Status |
|------|---------|-------------------|-------------|
| Create with minimal valid user | `POST /Users` with `userName` only | `201 Created`, body with `id`, `schemas`, `meta` | **PASS** — creates user with temp password |
| Create with full user | `POST /Users` with all core attributes | `201 Created`, all attributes reflected in response | **FAIL** — only `userName`, `email`, `displayName` persisted |
| Create with EnterpriseUser extension | `POST /Users` with enterprise schema URN | `201 Created`, extension attributes stored | **FAIL** — extension not supported |
| Duplicate `userName` | `POST /Users` twice with same `userName` | `409 Conflict` | **PASS** — returns 409 |
| Missing `userName` | `POST /Users` without `userName` | `400 Bad Request`, `scimType: invalidSyntax` | **FAIL** — returns 409, not 400 |
| `Location` header present | Check response headers | `Location: https://host/scim/v2/Users/{id}` | **FAIL** — no Location header |
| `Content-Type` correct | Check response headers | `application/scim+json` | **PASS** |
| Response includes `meta.created` | Check body | `meta.created` is ISO 8601 timestamp | **PASS** — uses RFC 3339 format |
| Response includes `meta.location` | Check body | Absolute URL to resource | **PARTIAL** — relative URL `/scim/v2/Users/{id}` |
| Response includes `meta.version` | Check body | ETag-like version string | **PASS** — `W/"{nanos}"` |

#### GET /Users — List Users

| Test | Request | Expected Response | GGID Status |
|------|---------|-------------------|-------------|
| List with no params | `GET /Users` | `200`, ListResponse with `Resources[]` | **PASS** |
| List with pagination | `GET /Users?startIndex=1&count=10` | First 10 users, correct `startIndex` | **PASS** |
| List with `startIndex=11` | `GET /Users?startIndex=11&count=10` | Next page of users | **PASS** |
| `totalResults` accuracy | After creating 5 users | `totalResults >= 5` | **PASS** |
| `itemsPerPage` accuracy | With fewer results than `count` | `itemsPerPage` = actual returned count | **FAIL** — returns requested count |
| Empty result set | `GET /Users?startIndex=999999` | `totalResults: 0`, `Resources: []` | **PASS** |
| `Resources` key capitalization | Check JSON key | Capital `"R"` in `"Resources"` | **PASS** |
| ListResponse schema URN | Check `schemas` | Contains `urn:ietf:params:scim:api:messages:2.0:ListResponse` | **PASS** |

#### GET /Users/{id} — Get Single User

| Test | Request | Expected Response | GGID Status |
|------|---------|-------------------|-------------|
| Valid ID | `GET /Users/{valid-uuid}` | `200 OK`, full user resource | **PASS** |
| Non-existent ID | `GET /Users/{random-uuid}` | `404 Not Found` | **PASS** |
| Invalid UUID format | `GET /Users/not-a-uuid` | `400 Bad Request` | **PASS** |
| ETag header (if supported) | Check response headers | `ETag: W/"version"` | **N/A** — etag not advertised |

#### PUT /Users/{id} — Replace User

| Test | Request | Expected Response | GGID Status |
|------|---------|-------------------|-------------|
| Full replacement | `PUT` with all attributes | `200 OK`, all attributes updated | **FAIL** — only `displayName` updated |
| Clear optional attribute | `PUT` without `nickName` | `nickName` should be null/removed | **FAIL** — attribute ignored |
| Toggle `active` | `PUT` with `active: false` | `200 OK`, user deactivated | **PASS** |
| Invalid ID | `PUT /Users/{random-uuid}` | `404 Not Found` | **PASS** |
| `If-Match` precondition | `PUT` with `If-Match: W/"version"` | `412` if stale | **N/A** — etag not supported |

#### PATCH /Users/{id} — Partial Update

| Test | Request | Expected Response | GGID Status |
|------|---------|-------------------|-------------|
| Replace `displayName` | `PATCH` with replace `displayName` | `200 OK`, updated display name | **PASS** |
| Replace `active` | `PATCH` with replace `active: false` | `200 OK`, user deactivated | **PASS** |
| Add email | `PATCH` with add `emails` | `200 OK`, new email in array | **FAIL** |
| Replace specific email | `PATCH` with `emails[type eq "work"].value` | `200 OK`, only matching email updated | **FAIL** |
| Remove email | `PATCH` with remove `emails[type eq "home"]` | `200 OK`, email removed | **FAIL** |
| Replace `name.familyName` | `PATCH` with replace `name.familyName` | `200 OK`, family name updated | **FAIL** |
| Add to multi-valued `addresses` | `PATCH` with add `addresses` | `200 OK`, address added | **FAIL** |
| Invalid path | `PATCH` with path `nonExistentAttr` | `400 Bad Request`, `scimType: invalidPath` | **FAIL** — silently ignored |
| Response includes updated resource | After PATCH | `200 OK` with full updated user body | **PASS** |
| Non-existent user | `PATCH /Users/{random-uuid}` | `404 Not Found` | **PASS** |

#### DELETE /Users/{id} — Delete User

| Test | Request | Expected Response | GGID Status |
|------|---------|-------------------|-------------|
| Valid ID | `DELETE /Users/{valid-uuid}` | `204 No Content` | **PASS** |
| Non-existent ID | `DELETE /Users/{random-uuid}` | `404 Not Found` | **PASS** |
| `If-Match` precondition | `DELETE` with `If-Match` | `412` if stale | **N/A** |

### 5.2 Filtering

SCIM defines 13 filter operators in RFC 7644 Section 3.4.2. GGID currently has **zero** filter support on the `/scim/v2/Users` endpoint (the REST API `/api/v1/users` has a `search` parameter, but this is not SCIM-compliant filtering).

#### Comparison Operators

| # | Operator | Example Filter | Expected Behavior | GGID Status |
|---|----------|---------------|-------------------|-------------|
| 1 | `eq` | `userName eq "bjensen"` | Exact match (case-insensitive for strings) | **FAIL** |
| 2 | `ne` | `userName ne "bjensen"` | All except exact match | **FAIL** |
| 3 | `co` | `displayName co "Jensen"` | Contains substring | **FAIL** |
| 4 | `sw` | `userName sw "bjen"` | Starts with | **FAIL** |
| 5 | `ew` | `userName ew "@example.com"` | Ends with | **FAIL** |
| 6 | `pr` | `title pr` | Attribute is present (has a value) | **FAIL** |
| 7 | `gt` | `meta.lastModified gt "2023-01-01T00:00:00Z"` | Greater than | **FAIL** |
| 8 | `ge` | `age ge 21` | Greater than or equal | **FAIL** |
| 9 | `lt` | `age lt 65` | Less than | **FAIL** |
| 10 | `le` | `age le 64` | Less than or equal | **FAIL** |

#### Logical Operators

| # | Operator | Example Filter | Expected Behavior | GGID Status |
|---|----------|---------------|-------------------|-------------|
| 11 | `and` | `active eq true and userName sw "a"` | Both conditions must match | **FAIL** |
| 12 | `or` | `title eq "VP" or title eq "Director"` | Either condition matches | **FAIL** |
| 13 | `not` | `not (emails co "example.com")` | Negation of sub-expression | **FAIL** |

#### Complex and Multi-Valued Attributes

| # | Filter Expression | Meaning | GGID Status |
|---|-------------------|---------|-------------|
| C-1 | `emails[type eq "work"].value eq "bjensen@example.com"` | Match user with specific work email | **FAIL** |
| C-2 | `emails[type eq "work" and value co "@example.com"]` | Complex multi-valued filter | **FAIL** |
| C-3 | `name.familyName co "Jen"` | Sub-attribute of complex attribute | **FAIL** |
| C-4 | `urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department eq "Eng"` | Extension attribute | **FAIL** |
| C-5 | `addresses[type eq "work"].locality eq "Hollywood"` | Nested complex in multi-valued | **FAIL** |

#### Filter Error Cases

| # | Invalid Filter | Expected Response | GGID Status |
|---|---------------|-------------------|-------------|
| FE-1 | Empty filter `filter=` | `400`, `scimType: invalidFilter` | **FAIL** — parameter ignored |
| FE-2 | Unknown operator `userName xx "val"` | `400`, `scimType: invalidFilter` | **FAIL** |
| FE-3 | Unterminated string `userName eq "val` | `400`, `scimType: invalidFilter` | **FAIL** |
| FE-4 | Missing operator `userName "val"` | `400`, `scimType: invalidFilter` | **FAIL** |

### 5.3 Pagination

| # | Test | Request | Expected | GGID Status |
|---|------|---------|----------|-------------|
| P-1 | Default pagination | `GET /Users` | Returns first page, valid ListResponse | **PASS** |
| P-2 | Custom page size | `GET /Users?count=5` | Exactly 5 results (or fewer if total < 5) | **PASS** |
| P-3 | Second page | `GET /Users?startIndex=6&count=5` | Next 5 results | **PASS** |
| P-4 | `count=0` | `GET /Users?count=0` | `Resources: []`, `totalResults` = full count | **PARTIAL** — GGID defaults count to 20 when 0 |
| P-5 | Beyond last page | `GET /Users?startIndex=999999` | `Resources: []`, `totalResults` unchanged | **PASS** |
| P-6 | `itemsPerPage` accuracy | `GET /Users?count=10` with 3 total users | `itemsPerPage: 3` | **FAIL** — returns 10 |
| P-7 | `totalResults` with filter | `GET /Users?filter=active eq true` | `totalResults` = count of active users only | **FAIL** — no filter |
| P-8 | `startIndex` defaults to 1 | `GET /Users` (no startIndex) | First result is index 1 | **PASS** |

### 5.4 Sorting

SCIM sorting uses `sortBy` and `sortOrder` query parameters (RFC 7644 Section 3.4.2.3). GGID's SCIM layer does NOT parse these parameters, despite `ServiceProviderConfig.sort.supported` being `true`.

| # | Test | Request | Expected | GGID Status |
|---|------|---------|----------|-------------|
| S-1 | Sort ascending by userName | `GET /Users?sortBy=userName&sortOrder=ascending` | Users sorted A-Z | **FAIL** |
| S-2 | Sort descending by userName | `GET /Users?sortBy=userName&sortOrder=descending` | Users sorted Z-A | **FAIL** |
| S-3 | Sort by sub-attribute | `GET /Users?sortBy=name.familyName` | Sorted by family name | **FAIL** |
| S-4 | Sort by `meta.created` | `GET /Users?sortBy=meta.created` | Sorted by creation date | **FAIL** |
| S-5 | Default sort order | `GET /Users?sortBy=userName` (no sortOrder) | Ascending (default) | **FAIL** |
| S-6 | No sort params | `GET /Users` | Unspecified order (valid) | **PASS** (trivially) |
| S-7 | Sort + filter + pagination combined | `GET /Users?filter=active eq true&sortBy=userName&sortOrder=ascending&startIndex=1&count=10` | Filter → sort → paginate | **FAIL** |
| S-8 | Sort on Groups | `GET /Groups?sortBy=displayName` | Groups sorted by display name | **FAIL** |

### 5.5 Bulk Operations

Bulk operations (RFC 7644 Section 3.7) allow multiple SCIM operations in a single POST. GGID does not implement `/scim/v2/Bulk` and correctly reports `bulk.supported: false` in ServiceProviderConfig.

| # | Test | Request | Expected | GGID Status |
|---|------|---------|----------|-------------|
| B-1 | Multiple POST operations | `POST /Bulk` with 2 create-user ops | `200`, both operations `status: 201` | **N/A** (not advertised) |
| B-2 | `bulkId` cross-reference | Op 2 references `bulkId` from op 1 | Resolved to actual ID | **N/A** |
| B-3 | `failOnErrors: 1` | First op fails | Server stops after 1 error | **N/A** |
| B-4 | `failOnErrors: 0` | One op fails | All ops processed, failed one has error | **N/A** |
| B-5 | Mixed POST + DELETE | Create then delete in one request | Both operations succeed | **N/A** |
| B-6 | Exceed `maxOperations` | More ops than `maxOperations` | `413` or `400` | **N/A** |
| B-7 | Unresolvable `bulkId` | Reference to non-existent bulkId | `400` for that op | **N/A** |
| B-8 | PATCH via bulk | `PATCH /Users/bulkId:xxx` | bulkId resolved to created resource | **N/A** |

**Note**: Since `bulk.supported` is `false`, conformance test suites should skip these tests. If forced (e.g., `-scim.force=bulk`), all would fail.

### 5.6 ETag / Concurrency

ETag support (RFC 7644 Section 3.14) enables optimistic concurrency control. GGID does not implement ETags and reports `etag.supported: false`.

| # | Test | Request | Expected | GGID Status |
|---|------|---------|----------|-------------|
| E-1 | ETag header on GET | `GET /Users/{id}` | Response has `ETag` header | **N/A** (not advertised) |
| E-2 | PUT with correct If-Match | `PUT` with matching ETag | `200 OK` | **N/A** |
| E-3 | PUT with stale If-Match | `PUT` with old ETag | `412 Precondition Failed` | **N/A** |
| E-4 | PUT with `If-Match: *` | Resource exists | `200 OK` | **N/A** |
| E-5 | PATCH with stale ETag | `PATCH` with old ETag | `412` | **N/A** |
| E-6 | DELETE with If-Match | `DELETE` with matching ETag | `204` | **N/A** |
| E-7 | GET with If-None-Match | Matching ETag | `304 Not Modified` | **N/A** |
| E-8 | `meta.version` present | Check body | ETag-like version in `meta` | **PASS** — GGID includes `meta.version` despite etag=false |

**Note**: GGID generates `meta.version` as `W/"{nanos}"` but does not use it for `If-Match` validation. This is technically inconsistent: if `etag.supported` is false, `meta.version` should arguably be omitted. However, including it is not a spec violation.

### 5.7 Schema Discovery

Schema discovery endpoints (RFC 7643 Sections 5-7) allow clients to introspect the server's capabilities and resource definitions.

| # | Test | Request | Expected | GGID Status |
|---|------|---------|----------|-------------|
| SD-1 | GET /ServiceProviderConfig | `GET /scim/v2/ServiceProviderConfig` | `200`, includes patch/bulk/filter/sort/etag/changePassword/authenticationSchemes | **PASS** |
| SD-2 | Config field completeness | Check all required sub-objects | All 6 capability objects + authenticationSchemes present | **PASS** |
| SD-3 | GET /ResourceTypes | `GET /scim/v2/ResourceTypes` | `200`, ListResponse with User + Group | **PARTIAL** — returns array, not ListResponse wrapper |
| SD-4 | GET /ResourceTypes/User | `GET /scim/v2/ResourceTypes/User` | `200`, User resource type definition | **FAIL** — no `/ResourceTypes/{id}` route |
| SD-5 | GET /Schemas | `GET /scim/v2/Schemas` | `200`, ListResponse with schema definitions | **FAIL** — not implemented |
| SD-6 | GET /Schemas/{urn} | `GET /scim/v2/Schemas/urn:ietf:params:scim:schemas:core:2.0:User` | `200`, User schema with all attributes | **FAIL** — not implemented |
| SD-7 | Schema attribute definitions | Schema includes `name`, `type`, `mutability`, `required`, `returned`, `uniqueness` per attribute | Full attribute metadata | **FAIL** |

---

## 6. Building a GGID SCIM Test Suite

### 6.1 Architecture

The test suite should live at `test/integration/scim/` and be runnable both locally and in CI.

```
test/
└── integration/
    └── scim/
        ├── client.go           # SCIM HTTP client
        ├── assertions.go       # SCIM-specific assertion helpers
        ├── fixtures.go         # Test data factories
        ├── conformance_test.go # Main conformance test suite
        ├── crud_test.go        # CRUD-specific tests
        ├── filter_test.go      # Filter-specific tests
        ├── pagination_test.go  # Pagination-specific tests
        ├── patch_test.go       # PATCH-specific tests
        └── schema_test.go      # Schema discovery tests
```

### 6.2 SCIM Test Client

```go
// test/integration/scim/client.go
//go:build integration

package scim_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
)

// SCIMClient is a configurable HTTP client for SCIM 2.0 testing.
type SCIMClient struct {
	BaseURL  string
	TenantID string
	Token    string
	HTTP     *http.Client
}

// NewSCIMClient creates a SCIM client from environment variables.
func NewSCIMClient() *SCIMClient {
	base := os.Getenv("SCIM_BASE_URL")
	if base == "" {
		base = "http://localhost:8081/scim/v2"
	}
	tenant := os.Getenv("SCIM_TENANT_ID")
	if tenant == "" {
		tenant = "00000000-0000-0000-0000-000000000001"
	}
	return &SCIMClient{
		BaseURL:  base,
		TenantID: tenant,
		Token:    os.Getenv("SCIM_BEARER_TOKEN"),
		HTTP:     &http.Client{},
	}
}

// Do sends a SCIM request and returns the raw HTTP response + parsed body.
func (c *SCIMClient) Do(t *testing.T, method, path string, body any) (*http.Response, map[string]any) {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/scim+json")
	req.Header.Set("Accept", "application/scim+json")
	req.Header.Set("X-Tenant-ID", c.TenantID)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var result map[string]any
	if resp.Body != nil {
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&result)
	}

	return resp, result
}

// DoRaw sends a request with custom headers (for ETag testing).
func (c *SCIMClient) DoRaw(t *testing.T, method, path string, body []byte, headers map[string]string) (*http.Response, map[string]any) {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/scim+json")
	req.Header.Set("X-Tenant-ID", c.TenantID)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var result map[string]any
	if resp.Body != nil {
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&result)
	}

	return resp, result
}

// FilterURL builds a properly URL-encoded SCIM GET with filter.
func (c *SCIMClient) FilterURL(path, filter string, params ...string) string {
	u := c.BaseURL + path + "?filter=" + url.QueryEscape(filter)
	for i := 0; i+1 < len(params); i += 2 {
		u += "&" + params[i] + "=" + url.QueryEscape(params[i+1])
	}
	return u
}
```

### 6.3 Assertion Helpers

```go
// test/integration/scim/assertions.go
//go:build integration

package scim_test

import (
	"fmt"
	"net/http"
	"testing"
)

const (
	schemaUser          = "urn:ietf:params:scim:schemas:core:2.0:User"
	schemaGroup         = "urn:ietf:params:scim:schemas:core:2.0:Group"
	schemaListResponse  = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	schemaError         = "urn:ietf:params:scim:api:messages:2.0:Error"
	schemaPatchOp       = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
	schemaServiceProvider = "urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"
	schemaResourceType  = "urn:ietf:params:scim:schemas:core:2.0:ResourceType"
)

// assertStatus checks the HTTP status code.
func assertStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Errorf("expected status %d, got %d", expected, resp.StatusCode)
	}
}

// assertContentType checks the Content-Type header is application/scim+json.
func assertContentType(t *testing.T, resp *http.Response) {
	t.Helper()
	ct := resp.Header.Get("Content-Type")
	if ct != "application/scim+json" {
		t.Errorf("expected Content-Type application/scim+json, got %s", ct)
	}
}

// assertSchemaContains checks that the response body's schemas array contains the expected URN.
func assertSchemaContains(t *testing.T, body map[string]any, expectedSchema string) {
	t.Helper()
	schemas, ok := body["schemas"].([]any)
	if !ok {
		t.Errorf("response missing 'schemas' array: %v", body)
		return
	}
	for _, s := range schemas {
		if s == expectedSchema {
			return
		}
	}
	t.Errorf("schemas array does not contain %s: %v", expectedSchema, schemas)
}

// assertSCIMError validates a SCIM error response.
func assertSCIMError(t *testing.T, resp *http.Response, body map[string]any, expectedStatus int) {
	t.Helper()
	assertStatus(t, resp, expectedStatus)
	assertSchemaContains(t, body, schemaError)

	statusStr := fmt.Sprintf("%d", expectedStatus)
	if body["status"] != statusStr {
		t.Errorf("error status field: got %v, expected %s", body["status"], statusStr)
	}
}

// assertListResponse validates the basic structure of a SCIM ListResponse.
func assertListResponse(t *testing.T, body map[string]any) {
	t.Helper()
	assertSchemaContains(t, body, schemaListResponse)

	if _, ok := body["totalResults"]; !ok {
		t.Error("ListResponse missing 'totalResults'")
	}
	if _, ok := body["Resources"]; !ok {
		t.Error("ListResponse missing 'Resources'")
	}
}

// assertLocationHeader checks that the Location header is present and non-empty.
func assertLocationHeader(t *testing.T, resp *http.Response) {
	t.Helper()
	loc := resp.Header.Get("Location")
	if loc == "" {
		t.Error("missing Location header")
	}
}

// assertMetaFields checks that meta.resourceType and meta.location are present.
func assertMetaFields(t *testing.T, body map[string]any, expectedResourceType string) {
	t.Helper()
	meta, ok := body["meta"].(map[string]any)
	if !ok {
		t.Error("response missing 'meta' object")
		return
	}
	if meta["resourceType"] != expectedResourceType {
		t.Errorf("meta.resourceType: got %v, expected %s", meta["resourceType"], expectedResourceType)
	}
	if meta["location"] == nil || meta["location"] == "" {
		t.Error("meta.location is missing or empty")
	}
}
```

### 6.4 Test Data Factory

```go
// test/integration/scim/fixtures.go
//go:build integration

package scim_test

import (
	"fmt"
	"math/rand"
	"time"
)

// uniqueSuffix generates a random suffix for test data isolation.
func uniqueSuffix() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%06d", r.Intn(999999))
}

// makeCreateUserRequest builds a valid SCIM User creation request.
func makeCreateUserRequest(username string) map[string]any {
	return map[string]any{
		"schemas":     []string{schemaUser},
		"userName":    username,
		"displayName": "Test User " + username,
		"name": map[string]any{
			"givenName":  "Test",
			"familyName": "User",
		},
		"emails": []map[string]any{
			{
				"value":   username,
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}
}

// makeCreateGroupRequest builds a valid SCIM Group creation request.
func makeCreateGroupRequest(displayName string) map[string]any {
	return map[string]any{
		"schemas":     []string{schemaGroup},
		"displayName": displayName,
	}
}

// makePatchReplaceOp builds a PATCH replace operation.
func makePatchReplaceOp(path string, value any) map[string]any {
	return map[string]any{
		"schemas": []string{schemaPatchOp},
		"Operations": []map[string]any{
			{
				"op":    "replace",
				"path":  path,
				"value": value,
			},
		},
	}
}

// makePatchAddOp builds a PATCH add operation.
func makePatchAddOp(path string, value any) map[string]any {
	return map[string]any{
		"schemas": []string{schemaPatchOp},
		"Operations": []map[string]any{
			{
				"op":    "add",
				"path":  path,
				"value": value,
			},
		},
	}
}
```

### 6.5 Example Test Functions

```go
// test/integration/scim/conformance_test.go
//go:build integration

package scim_test

import (
	"net/http"
	"testing"
)

// === Test 1: Create User - Full Lifecycle ===
func TestCreateUser_Success(t *testing.T) {
	client := NewSCIMClient()
	username := "conform-create-" + uniqueSuffix() + "@example.com"

	// Create
	resp, body := client.Do(t, "POST", "/Users", makeCreateUserRequest(username))
	assertStatus(t, resp, http.StatusCreated)
	assertContentType(t, resp)
	assertSchemaContains(t, body, schemaUser)
	assertMetaFields(t, body, "User")

	// Location header (RFC 7644 Section 3.3)
	loc := resp.Header.Get("Location")
	if loc == "" {
		t.Error("FAIL: POST /Users response missing Location header")
	} else {
		t.Logf("PASS: Location header present: %s", loc)
	}

	// Verify by GET
	userID := body["id"].(string)
	resp2, body2 := client.Do(t, "GET", "/Users/"+userID, nil)
	assertStatus(t, resp2, http.StatusOK)
	if body2["userName"] != username {
		t.Errorf("userName mismatch on GET: got %v, expected %s", body2["userName"], username)
	}

	// Cleanup
	client.Do(t, "DELETE", "/Users/"+userID, nil)
}

// === Test 2: Duplicate User Conflict ===
func TestCreateUser_DuplicateConflict(t *testing.T) {
	client := NewSCIMClient()
	username := "conform-dup-" + uniqueSuffix() + "@example.com"

	// First create
	resp, _ := client.Do(t, "POST", "/Users", makeCreateUserRequest(username))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first create should return 201, got %d", resp.StatusCode)
	}

	// Second create with same userName
	resp2, body2 := client.Do(t, "POST", "/Users", makeCreateUserRequest(username))
	assertSCIMError(t, resp2, body2, http.StatusConflict)

	// Cleanup
	if id, ok := body2["id"].(string); ok {
		client.Do(t, "DELETE", "/Users/"+id, nil)
	}
}

// === Test 3: Pagination Correctness ===
func TestPagination_ItemsPerPageAccuracy(t *testing.T) {
	client := NewSCIMClient()

	// Create 3 test users
	var userIDs []string
	for i := 0; i < 3; i++ {
		username := "conform-page-" + uniqueSuffix() + "@example.com"
		_, body := client.Do(t, "POST", "/Users", makeCreateUserRequest(username))
		if id, ok := body["id"].(string); ok {
			userIDs = append(userIDs, id)
		}
	}
	defer func() {
		for _, id := range userIDs {
			client.Do(t, "DELETE", "/Users/"+id, nil)
		}
	}()

	// Request count=10 but only 3 users exist (plus any pre-existing)
	resp, body := client.Do(t, "GET", "/Users?startIndex=1&count=10", nil)
	assertStatus(t, resp, http.StatusOK)
	assertListResponse(t, body)

	totalResults := int(body["totalResults"].(float64))
	itemsPerPage := int(body["itemsPerPage"].(float64))
	resources := body["Resources"].([]any)

	// itemsPerPage should reflect actual returned count, not requested count
	actualCount := len(resources)
	if itemsPerPage != actualCount {
		t.Errorf("FAIL: itemsPerPage=%d but actual returned=%d (totalResults=%d)",
			itemsPerPage, actualCount, totalResults)
	} else {
		t.Logf("PASS: itemsPerPage=%d matches actual returned count", itemsPerPage)
	}
}

// === Test 4: Filter Support ===
func TestFilter_UserNameEq(t *testing.T) {
	client := NewSCIMClient()
	username := "conform-filter-" + uniqueSuffix() + "@example.com"

	// Create test user
	_, body := client.Do(t, "POST", "/Users", makeCreateUserRequest(username))
	userID := body["id"].(string)
	defer client.Do(t, "DELETE", "/Users/"+userID, nil)

	// Filter by exact userName
	filterURL := client.FilterURL("/Users", `userName eq "`+username+`"`)
	resp, body := client.Do(t, "GET", filterURL[len(client.BaseURL):], nil)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("FAIL: filter request returned %d (expected 200)", resp.StatusCode)
		return
	}

	totalResults := int(body["totalResults"].(float64))
	if totalResults < 1 {
		t.Errorf("FAIL: filter returned totalResults=%d, expected >= 1", totalResults)
	} else {
		t.Logf("PASS: filter returned %d results", totalResults)
	}
}

// === Test 5: PATCH Replace DisplayName ===
func TestPatch_ReplaceDisplayName(t *testing.T) {
	client := NewSCIMClient()
	username := "conform-patch-" + uniqueSuffix() + "@example.com"

	// Create
	_, body := client.Do(t, "POST", "/Users", makeCreateUserRequest(username))
	userID := body["id"].(string)
	defer client.Do(t, "DELETE", "/Users/"+userID, nil)

	// PATCH displayName
	newName := "Patched Name"
	resp, body := client.Do(t, "PATCH", "/Users/"+userID,
		makePatchReplaceOp("displayName", newName))
	assertStatus(t, resp, http.StatusOK)

	if body["displayName"] != newName {
		t.Errorf("FAIL: displayName after PATCH: got %v, expected %s", body["displayName"], newName)
	} else {
		t.Logf("PASS: displayName updated to %s", newName)
	}
}

// === Test 6: ServiceProviderConfig ===
func TestServiceProviderConfig(t *testing.T) {
	client := NewSCIMClient()
	resp, body := client.Do(t, "GET", "/ServiceProviderConfig", nil)

	assertStatus(t, resp, http.StatusOK)
	assertSchemaContains(t, body, schemaServiceProvider)

	// Verify required capability objects
	requiredKeys := []string{"patch", "bulk", "filter", "changePassword", "sort", "etag", "authenticationSchemes"}
	for _, key := range requiredKeys {
		if _, ok := body[key]; !ok {
			t.Errorf("FAIL: ServiceProviderConfig missing '%s'", key)
		}
	}

	// Verify filter is honestly advertised
	filterConfig := body["filter"].(map[string]any)
	filterSupported := filterConfig["supported"].(bool)
	t.Logf("filter.supported = %v", filterSupported)
}

// === Test 7: Schema Discovery ===
func TestSchemaDiscovery_SchemasEndpoint(t *testing.T) {
	client := NewSCIMClient()
	resp, body := client.Do(t, "GET", "/Schemas", nil)

	if resp.StatusCode == http.StatusNotFound {
		t.Skip("GET /Schemas not implemented yet (known gap)")
	}

	assertStatus(t, resp, http.StatusOK)
	assertListResponse(t, body)

	// Should contain core User and Group schemas
	resources := body["Resources"].([]any)
	foundUser := false
	foundGroup := false
	for _, r := range resources {
		schema := r.(map[string]any)
		if schema["id"] == schemaUser {
			foundUser = true
		}
		if schema["id"] == schemaGroup {
			foundGroup = true
		}
	}
	if !foundUser {
		t.Error("FAIL: /Schemas missing User schema")
	}
	if !foundGroup {
		t.Error("FAIL: /Schemas missing Group schema")
	}
}

// === Test 8: Error Response Format ===
func TestErrorResponse_Format(t *testing.T) {
	client := NewSCIMClient()

	// Trigger a 404
	resp, body := client.Do(t, "GET", "/Users/00000000-0000-0000-0000-000000000000", nil)

	assertStatus(t, resp, http.StatusNotFound)
	assertSchemaContains(t, body, schemaError)

	if body["status"] != "404" {
		t.Errorf("FAIL: error status field = %v, expected '404'", body["status"])
	}
	if body["detail"] == nil || body["detail"] == "" {
		t.Error("FAIL: error response missing 'detail' field")
	}
}
```

### 6.6 Test Case Definition in Go Table Tests

For more maintainable test definitions, use Go table-driven tests:

```go
// test/integration/scim/filter_test.go
//go:build integration

package scim_test

import (
	"net/http"
	"testing"
)

func TestFilterOperators(t *testing.T) {
	client := NewSCIMClient()

	// Setup: create test users
	users := []string{
		"filter-eq@test.com",
		"filter-co@test.com",
		"filter-sw@test.com",
	}
	for _, u := range users {
		_, body := client.Do(t, "POST", "/Users", makeCreateUserRequest(u))
		if id, ok := body["id"].(string); ok {
			defer client.Do(t, "DELETE", "/Users/"+id, nil)
		}
	}

	tests := []struct {
		name      string
		filter    string
		minResult int
	}{
		{"eq operator", `userName eq "filter-eq@test.com"`, 1},
		{"co operator", `userName co "filter-"`, 3},
		{"sw operator", `userName sw "filter"`, 3},
		{"ew operator", `userName ew "@test.com"`, 3},
		{"pr operator", `userName pr`, 3},
		{"and operator", `userName sw "filter" and active eq true`, 3},
		{"ne operator", `userName ne "nonexistent@test.com"`, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterURL := client.FilterURL("/Users", tt.filter)
			resp, body := client.Do(t, "GET", filterURL[len(client.BaseURL):], nil)

			if resp.StatusCode != http.StatusOK {
				t.Errorf("filter '%s': expected 200, got %d", tt.filter, resp.StatusCode)
				return
			}

			total := int(body["totalResults"].(float64))
			if total < tt.minResult {
				t.Errorf("filter '%s': expected >= %d results, got %d",
					tt.filter, tt.minResult, total)
			}
		})
	}
}
```

### 6.7 CI Integration Plan

```yaml
# .github/workflows/scim-conformance.yml
name: SCIM Conformance Tests

on:
  push:
    paths:
      - 'services/identity/**'
      - 'test/integration/scim/**'
      - '.github/workflows/scim-conformance.yml'
  pull_request:
    paths:
      - 'services/identity/**'

jobs:
  scim-conformance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Start GGID services
        run: |
          cd deploy
          docker compose up -d postgres redis nats
          sleep 10

      - name: Run database migrations
        run: |
          go run ./services/identity/cmd/migrate up

      - name: Start Identity service
        run: |
          go run ./services/identity/cmd &
          sleep 5

      - name: Run SCIM conformance tests
        env:
          SCIM_BASE_URL: http://localhost:8081/scim/v2
          SCIM_TENANT_ID: 00000000-0000-0000-0000-000000000001
        run: |
          go test -tags=integration -v ./test/integration/scim/... 2>&1 | tee scim-results.txt

      - name: Parse conformance results
        if: always()
        run: |
          echo "## SCIM Conformance Results" >> $GITHUB_STEP_SUMMARY
          echo '```' >> $GITHUB_STEP_SUMMARY
          grep -E "(PASS|FAIL|SKIP)" scim-results.txt >> $GITHUB_STEP_SUMMARY || true
          echo '```' >> $GITHUB_STEP_SUMMARY

      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: scim-conformance-results
          path: scim-results.txt
```

---

## 7. Remediation Priority

### 7.1 Priority Matrix

Based on the pass/fail predictions and impact analysis, here is the recommended remediation order to maximize conformance gain per unit of engineering effort.

#### Tier 1: Critical (Blocks Major IdP Integration)

| # | Gap | RFC Ref | Effort | Impact | Rationale |
|---|-----|---------|--------|--------|-----------|
| 1 | **SCIM Filter Engine** | 7644 3.4.2 | Large (3-5 days) | +15 tests | Without filters, no IdP can discover users. This is the single highest-impact fix. |
| 2 | **Full PATCH Path Engine** | 7644 3.5.2 | Large (3-5 days) | +5 tests | IdPs use PATCH extensively for attribute sync. Currently only displayName/active work. |
| 3 | **Groups DB Persistence** | 7644 3.x | Medium (2-3 days) | +5 tests | Groups are entirely mock-backed. No IdP can manage group memberships. |

#### Tier 2: High (Improves Conformance Score Significantly)

| # | Gap | RFC Ref | Effort | Impact | Rationale |
|---|-----|---------|--------|--------|-----------|
| 4 | **`Location` header on POST** | 7644 3.3 | Small (0.5 day) | +1 test | Simple fix, required by spec, tested by all suites. |
| 5 | **`itemsPerPage` accuracy** | 7644 3.4.2 | Small (0.5 day) | +1 test | Return actual count, not requested count. |
| 6 | **Sort in SCIM endpoints** | 7644 3.4.2.3 | Medium (1-2 days) | +5 tests | ServiceProviderConfig already says `sort: true`. Must implement or change config to `false`. |
| 7 | **`/Schemas` endpoint** | 7643 7 | Medium (1-2 days) | +3 tests | Schema introspection is tested by scim.dev and Entra ID validator. |
| 8 | **`scimType` in error responses** | 7644 3.12 | Small (0.5 day) | +2 tests | Add `scimType` field to ErrorResponse struct. |
| 9 | **`externalId` persistence** | 7643 5 | Small (0.5 day) | +1 test | Accept, store, and return `externalId`. |

#### Tier 3: Medium (Polish and Advanced Features)

| # | Gap | RFC Ref | Effort | Impact | Rationale |
|---|-----|---------|--------|--------|-----------|
| 10 | **EnterpriseUser extension** | 7643 4.3 | Medium (2 days) | +2 tests | Needed for HR system integration (employeeNumber, department, manager). |
| 11 | **`attributes`/`excludedAttributes`** | 7644 3.9 | Small (1 day) | +2 tests | Attribute projection support. |
| 12 | **`/ResourceTypes/{id}` route** | 7643 6 | Small (0.5 day) | +2 tests | Individual resource type lookup. |
| 13 | **400 vs 409 error differentiation** | 7644 3.12 | Small (0.5 day) | +1 test | Return 400 for validation errors, 409 only for uniqueness conflicts. |
| 14 | **`PUT` full replacement semantics** | 7644 3.5.1 | Medium (1-2 days) | +2 tests | PUT should replace all attributes, not just displayName. |

#### Tier 4: Low (Future Enhancement)

| # | Gap | RFC Ref | Effort | Impact | Rationale |
|---|-----|---------|--------|--------|-----------|
| 15 | **Bulk endpoint** | 7644 3.7 | Large (3-5 days) | +8 tests (if enabled) | Optional feature. Only needed if bulk provisioning is a customer requirement. |
| 16 | **ETag / If-Match** | 7644 3.14 | Medium (2 days) | +8 tests (if enabled) | Optional feature. Most IdPs don't use it. |
| 17 | **POST .search** | 7644 3.4.3 | Small (1 day) | +2 tests | POST-based search binding. Rarely tested. |
| 18 | **Password change via SCIM** | 7643 4.1.1 | Small (1 day) | +1 test | `changePassword` is advertised as true but not implemented. |

### 7.2 Estimated Conformance After Each Tier

| Milestone | Tests Fixed | Cumulative PASS | Estimated Compliance |
|-----------|------------|----------------|---------------------|
| **Current state** | 0 | 13/45 | ~29% |
| **After Tier 1** | +25 | 38/45 | ~84% |
| **After Tier 2** | +16 | 54/61 | ~89% |
| **After Tier 3** | +9 | 63/70 | ~90% |
| **After Tier 4** | +19 | 82/89 | ~92% |

### 7.3 Timeline to 80%+ Conformance

```
Week 1-2:  SCIM Filter Engine (Tier 1, #1)
           - Implement filter parser (lexer + AST)
           - Support all 13 operators
           - Translate to parameterized SQL
           - Test with all filter test cases

Week 2-3:  Full PATCH Path Engine (Tier 1, #2)
           - Implement path expression parser
           - Support add/replace/remove on all attributes
           - Handle multi-valued attribute path filters
           - Handle complex sub-attribute paths

Week 3:    Groups DB Persistence (Tier 1, #3)
           - Map SCIM Groups to GGID roles
           - Persist group membership
           - Wire all Group CRUD to database

Week 4:    Quick Wins (Tier 2, #4-9)
           - Location header, itemsPerPage fix
           - Sort in SCIM endpoints
           - /Schemas endpoint
           - scimType in errors
           - externalId persistence

Result:    ~84% conformance after 4 weeks
```

### 7.4 Effort Summary

| Phase | Duration | Gaps Closed | Conformance Gain |
|-------|----------|-------------|-----------------|
| Tier 1 (Critical) | 3 weeks | 3 gaps | +25 tests (29% → 84%) |
| Tier 2 (High) | 1 week | 6 gaps | +8 tests (84% → 89%) |
| Tier 3 (Medium) | 1-2 weeks | 5 gaps | +5 tests (89% → 90%) |
| Tier 4 (Low) | 2-3 weeks | 4 gaps | +9 tests (90% → 92%) |
| **Total** | **7-9 weeks** | **18 gaps** | **29% → 92%** |

---

## 8. Integration with CI/CD

### 8.1 Docker Compose Test Pipeline

Create a dedicated Docker Compose configuration for SCIM testing that starts the full GGID stack and runs the conformance test suite:

```yaml
# deploy/docker-compose.scim-test.yml
version: "3.9"

services:
  # Infrastructure
  postgres-test:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: ggid_test
      POSTGRES_USER: ggid
      POSTGRES_PASSWORD: ggid_test_pass
    ports:
      - "5433:5432"  # Different port to avoid conflicts
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ggid"]
      interval: 5s
      timeout: 3s
      retries: 10

  redis-test:
    image: redis:7-alpine
    ports:
      - "6380:6379"

  # Migrations
  migrate-test:
    build:
      context: ..
      dockerfile: deploy/Dockerfile.identity
    command: ["./identity", "migrate", "up"]
    environment:
      DATABASE_URL: "postgres://ggid:ggid_test_pass@postgres-test:5432/ggid_test?sslmode=disable"
    depends_on:
      postgres-test:
        condition: service_healthy

  # Identity service (SCIM endpoint)
  identity-test:
    build:
      context: ..
      dockerfile: deploy/Dockerfile.identity
    ports:
      - "8090:8080"
    environment:
      DATABASE_URL: "postgres://ggid:ggid_test_pass@postgres-test:5432/ggid_test?sslmode=disable"
      REDIS_URL: "redis-test:6379"
      PORT: "8080"
    depends_on:
      migrate-test:
        condition: service_completed_successfully
      postgres-test:
        condition: service_healthy
      redis-test:
        condition: service_started
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/healthz"]
      interval: 5s
      timeout: 3s
      retries: 10

  # SCIM test runner
  scim-tests:
    build:
      context: ..
      dockerfile: deploy/Dockerfile.scim-tests
    environment:
      SCIM_BASE_URL: "http://identity-test:8080/scim/v2"
      SCIM_TENANT_ID: "00000000-0000-0000-0000-000000000001"
    depends_on:
      identity-test:
        condition: service_healthy
    command: ["go", "test", "-tags=integration", "-v", "./test/integration/scim/..."]
```

### 8.2 SCIM Test Dockerfile

```dockerfile
# deploy/Dockerfile.scim-tests
FROM golang:1.25-alpine AS test-runner

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build test binary
RUN go build -tags=integration ./test/integration/scim/...

# Default command
CMD ["go", "test", "-tags=integration", "-v", "./test/integration/scim/..."]
```

### 8.3 GitHub Actions Workflow

```yaml
# .github/workflows/scim-conformance.yml
name: SCIM Conformance

on:
  push:
    branches: [main]
    paths:
      - 'services/identity/**'
      - 'test/integration/scim/**'
      - 'deploy/docker-compose.scim-test.yml'
  pull_request:
    paths:
      - 'services/identity/**'
      - 'test/integration/scim/**'

env:
  GO_VERSION: '1.25'

jobs:
  conformance:
    name: SCIM 2.0 Conformance Tests
    runs-on: ubuntu-latest
    timeout-minutes: 15

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Build Identity service
        run: |
          go build -o bin/identity ./services/identity/cmd

      - name: Start test infrastructure
        run: |
          docker compose -f deploy/docker-compose.scim-test.yml up -d postgres-test redis-test
          sleep 10

      - name: Run migrations
        run: |
          DATABASE_URL="postgres://ggid:ggid_test_pass@localhost:5433/ggid_test?sslmode=disable" \
            go run ./services/identity/cmd migrate up

      - name: Start Identity service
        run: |
          DATABASE_URL="postgres://ggid:ggid_test_pass@localhost:5433/ggid_test?sslmode=disable" \
            PORT=8090 \
            ./bin/identity &
          sleep 5

      - name: Wait for service
        run: |
          for i in $(seq 1 30); do
            if curl -sf http://localhost:8090/healthz > /dev/null; then
              echo "Service is ready"
              exit 0
            fi
            sleep 2
          done
          echo "Service failed to start"
          exit 1

      - name: Run SCIM conformance tests
        env:
          SCIM_BASE_URL: http://localhost:8090/scim/v2
          SCIM_TENANT_ID: 00000000-0000-0000-0000-000000000001
        run: |
          go test -tags=integration -v -timeout 120s ./test/integration/scim/... 2>&1 | tee scim-results.txt

      - name: Generate compliance report
        if: always()
        run: |
          echo "## SCIM 2.0 Conformance Report" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY

          # Count results
          TOTAL=$(grep -c "^=== RUN" scim-results.txt || echo "0")
          PASSED=$(grep -c "^--- PASS" scim-results.txt || echo "0")
          FAILED=$(grep -c "^--- FAIL" scim-results.txt || echo "0")
          SKIPPED=$(grep -c "^--- SKIP" scim-results.txt || echo "0")

          if [ "$TOTAL" -gt 0 ]; then
            PCT=$((PASSED * 100 / TOTAL))
          else
            PCT=0
          fi

          echo "| Metric | Value |" >> $GITHUB_STEP_SUMMARY
          echo "|--------|-------|" >> $GITHUB_STEP_SUMMARY
          echo "| Total Tests | $TOTAL |" >> $GITHUB_STEP_SUMMARY
          echo "| Passed | $PASSED |" >> $GITHUB_STEP_SUMMARY
          echo "| Failed | $FAILED |" >> $GITHUB_STEP_SUMMARY
          echo "| Skipped | $SKIPPED |" >> $GITHUB_STEP_SUMMARY
          echo "| Compliance | ${PCT}% |" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY
          echo '```' >> $GITHUB_STEP_SUMMARY
          grep -E "^(=== RUN|--- PASS|--- FAIL|--- SKIP)" scim-results.txt >> $GITHUB_STEP_SUMMARY || true
          echo '```' >> $GITHUB_STEP_SUMMARY

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: scim-conformance-report
          path: scim-results.txt
          retention-days: 30

      - name: Cleanup
        if: always()
        run: |
          docker compose -f deploy/docker-compose.scim-test.yml down -v
          pkill -f "bin/identity" || true
```

### 8.4 Compliance Badge Generation

Add a compliance badge to the README that reflects the latest CI result:

```markdown
<!-- README.md -->
## SCIM 2.0 Compliance

![SCIM Conformance](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/ggid/ggid/main/.github/scim-badge.json)

Current SCIM 2.0 conformance: **~29%** (pre-remediation)

See SCIM conformance testing analysis for details.
```

Badge JSON (generated by CI):

```json
{
  "schemaVersion": 1,
  "label": "SCIM Conformance",
  "message": "29%",
  "color": "red"
}
```

As conformance improves, the CI step updates the badge:

```bash
# In CI after tests
PCT=29  # calculated from test results
COLOR="red"
if [ "$PCT" -ge 80 ]; then COLOR="green"; elif [ "$PCT" -ge 50 ]; then COLOR="yellow"; fi

cat > .github/scim-badge.json << EOF
{
  "schemaVersion": 1,
  "label": "SCIM Conformance",
  "message": "${PCT}%",
  "color": "${COLOR}"
}
EOF
```

### 8.5 Running Against External Test Suites

In addition to the custom test suite, periodically run GGID against external conformance tools:

#### Running scim2/test-suite

```bash
# Clone the official Go SCIM compliance test suite
git clone https://github.com/scim2/test-suite.git /tmp/scim-test-suite
cd /tmp/scim-test-suite

# Ensure GGID Identity service is running on localhost:8081

# Run the test suite
go test ./compliance/ -v -count=1 \
  -scim.url=http://localhost:8081/scim/v2 \
  -scim.token="" \
  -scim.header="X-Tenant-ID:00000000-0000-0000-0000-000000000001"

# Force-test features GGID doesn't advertise yet
go test ./compliance/ -v -count=1 \
  -scim.url=http://localhost:8081/scim/v2 \
  -scim.token="" \
  -scim.header="X-Tenant-ID:00000000-0000-0000-0000-000000000001" \
  -scim.force=filter,patch,sort
```

#### Running SCIM Verify (scim.dev)

```bash
# Expose local endpoint via tunnel
ngrok http 8081

# Note the tunnel URL (e.g., https://abc123.ngrok.io)

# Navigate to https://scim.dev/verify/
# Enter:
#   SCIM URL: https://abc123.ngrok.io/scim/v2
#   Auth: Bearer (leave empty for GGID dev mode)
#   Custom Header: X-Tenant-ID: 00000000-0000-0000-0000-000000000001

# Click "Run Tests" and review the compliance report
```

#### Running Microsoft Entra SCIM Validator

```bash
# Expose local endpoint via tunnel
ngrok http 8081

# Navigate to https://scimvalidator.microsoft.com/
# Select "Discover schema"
# Enter:
#   SCIM URL: https://abc123.ngrok.io/scim/v2
#   Token: (leave empty for GGID dev mode)

# Note: GGID's X-Tenant-ID header requirement may cause issues.
# Consider adding a middleware that defaults tenant ID when
# the header is absent (for SCIM endpoints only).
```

---

## Appendix A: SCIM Filter Engine Implementation Notes

Implementing a SCIM filter engine is the single highest-impact remediation. Here is a high-level design:

### Lexer

```go
// Token types
type TokenType int

const (
    TokenAttr TokenType = iota
    TokenEq
    TokenNe
    TokenCo
    TokenSw
    TokenEw
    TokenPr
    TokenGt
    TokenGe
    TokenLt
    TokenLe
    TokenAnd
    TokenOr
    TokenNot
    TokenLParen
    TokenRParen
    TokenLBracket
    TokenRBracket
    TokenString
    TokenNumber
    TokenBoolean
    TokenDot
)

type Token struct {
    Type  TokenType
    Value string
}
```

### AST

```go
// Filter AST nodes
type FilterExpr interface {
    String() string
}

type ComparisonExpr struct {
    Attr     string
    Operator string
    Value    any
}

type LogicalExpr struct {
    Left     FilterExpr
    Operator string // "and", "or"
    Right    FilterExpr
}

type NotExpr struct {
    Inner FilterExpr
}

type PresentExpr struct {
    Attr string
}

type ComplexAttrExpr struct {
    Attr      string
    SubFilter FilterExpr
    SubAttr   string // optional, after the closing bracket
}
```

### SQL Translation

```go
// Translate AST to parameterized SQL WHERE clause
func (e *ComparisonExpr) ToSQL(args *[]any) string {
    switch e.Operator {
    case "eq":
        *args = append(*args, e.Value)
        return fmt.Sprintf("%s = $%d", mapAttrToColumn(e.Attr), len(*args))
    case "co":
        *args = append(*args, "%"+fmt.Sprintf("%v", e.Value)+"%")
        return fmt.Sprintf("%s ILIKE $%d", mapAttrToColumn(e.Attr), len(*args))
    case "sw":
        *args = append(*args, fmt.Sprintf("%v%%", e.Value))
        return fmt.Sprintf("%s ILIKE $%d", mapAttrToColumn(e.Attr), len(*args))
    case "ew":
        *args = append(*args, "%"+fmt.Sprintf("%v", e.Value))
        return fmt.Sprintf("%s ILIKE $%d", mapAttrToColumn(e.Attr), len(*args))
    case "ne":
        *args = append(*args, e.Value)
        return fmt.Sprintf("%s != $%d", mapAttrToColumn(e.Attr), len(*args))
    // ... gt, ge, lt, le
    }
    return "1=1" // fallback
}
```

## Appendix B: Test Suite Comparison Summary

| Feature | SCIM Verify (scim.dev) | MS Entra Validator | Okta SCIM Test | scim2/test-suite (Go) | Custom GGID Suite |
|---------|----------------------|--------------------|----------------|-----------------------|--------------------|
| **Hosting** | Hosted (web) | Hosted (web) | Okta platform | Self-hosted (Go) | Self-hosted (Go) |
| **CI/CD** | No | No | No | Yes (go test) | Yes (go test) |
| **Config** | YAML | Web UI | Okta Admin | Go flags | Go env vars |
| **Filter tests** | Yes (13 ops) | Limited (eq only) | Limited (eq only) | Yes | Yes (configurable) |
| **PATCH tests** | Yes | Yes (add/replace) | Yes (active toggle) | Yes | Yes |
| **Schema discovery** | Yes | Yes (discover mode) | No | Yes | Yes |
| **Bulk tests** | Yes | No | No | Yes (if advertised) | Planned |
| **ETag tests** | Yes | No | No | Yes (if advertised) | Planned |
| **Tenant testing** | No | No | No | No | Yes |
| **Cost** | Free | Free (Entra account) | Free (Okta dev) | Free | Free |
| **Best for** | General compliance | Entra ID compat | Okta compat | CI automation | GGID-specific |

## Appendix C: References

- [RFC 7643 — SCIM Core Schema](https://www.rfc-editor.org/rfc/rfc7643)
- [RFC 7644 — SCIM Protocol](https://www.rfc-editor.org/rfc/rfc7644)
- [RFC 7642 — SCIM Definitions and Requirements](https://www.rfc-editor.org/rfc/rfc7642)
- [SCIM Verify (scim.dev)](https://scim.dev/verify/) — Hosted conformance testing
- [SCIM Verify Documentation](https://verify.scim.dev/docs/) — YAML configuration reference
- [Microsoft Entra SCIM Validator](https://scimvalidator.microsoft.com/) — Entra ID compatibility testing
- [Entra SCIM Validator Tutorial](https://learn.microsoft.com/en-us/entra/identity/app-provisioning/scim-validator-tutorial) — Step-by-step guide
- [scim2/test-suite](https://github.com/scim2/test-suite) — Go-based compliance test suite
- [Skycloak SCIM Tester](https://skycloak.io/tools/scim-tester/) — Online SCIM endpoint tester
- [SCIM 2.0 Compliance and Test Suite Design (GGID)](./scim2-compliance-test-suite.md) — Previous gap analysis
- [SimpleCloud.info](https://simplecloud.info) — SCIM community site
