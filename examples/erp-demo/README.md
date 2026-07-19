# GGID ERP Demo

A complete Java Spring Boot ERP application demonstrating **fine-grained RBAC** with GGID IAM.

## Scenario

An ERP system with **8 modules** and **6 roles**, where different roles have different access levels to the same modules:

| Module | Sales Mgr | Warehouse Mgr | Finance Officer | HR Mgr | Production Mgr | ERP Admin |
|--------|:---------:|:------------:|:--------------:|:------:|:--------------:|:---------:|
| **Inventory** | read | read/write/delete | - | - | read | all |
| **Orders** | read/write/approve | read/write | read | - | - | all |
| **Customers** | read/write/delete | - | - | - | - | all |
| **Invoices** | - | - | read/write/approve/delete | - | - | all |
| **Payments** | - | - | read/write/approve | - | - | all |
| **Employees** | - | - | - | read/write/delete | - | all |
| **Production** | - | - | - | - | read/write/approve | all |
| **Reports** | read | read | read | read | read | read |

For example:
- **Sales Manager** and **Warehouse Manager** both access Orders, but Sales Manager can approve while Warehouse Manager can only ship.
- **Finance Officer** sees Invoices + Payments that no other role can access.
- **Everyone** can read Reports.

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    ERP App (Java/Spring)                  │
│                                                           │
│  ┌─────────────┐   ┌──────────────────┐   ┌────────────┐ │
│  │ AuthFilter  │──▶│ @RequirePermission│──▶│ Controller │ │
│  │ (JWT verify) │   │  (GGID PDP call) │   │  (business) │ │
│  └─────────────┘   └──────────────────┘   └────────────┘ │
│         │                    │                            │
│         ▼                    ▼                            │
└─────────┼────────────────────┼────────────────────────────┘
          │                    │
          ▼                    ▼
    ┌──────────────────────────────────┐
    │         GGID Gateway              │
    │  /api/v1/auth/login (auth)        │
    │  /api/v1/policies/check (RBAC)    │
    │  /oauth/jwks (JWT verification)    │
    └──────────────────────────────────┘
```

## How Permission Checking Works

Every protected endpoint has a `@RequirePermission` annotation:

```java
@PostMapping("/inventory/{id}/delete")
@RequirePermission(resource = "inventory", action = "delete")
public String deleteProduct(@PathVariable String id) {
    // Only warehouse_manager and erp_admin can reach here
    products.removeIf(p -> p.id.equals(id));
    return "redirect:/inventory";
}
```

The `PermissionAspect` interceptor calls GGID's Policy Decision Point (PDP):

```
POST /api/v1/policies/check
{
  "user_id": "<uuid from JWT>",
  "resource_type": "inventory",
  "action": "delete"
}
```

GGID evaluates the user's roles + permissions + ABAC policies and returns:
```json
{ "allowed": true, "reason": "allowed by RBAC role permission", "matched_by": "rbac" }
```

If the policy API is unreachable, a **local permission matrix** (`ErpPermissions.java`) provides fallback so the demo always works.

## Quick Start

### Prerequisites
- Java 17+
- Maven 3.9+
- GGID running (default: `https://ggid.iot2.win`)

### 1. Configure GGID

```bash
# Create roles, permissions, and demo users
GGID_URL=https://ggid.iot2.win \
ADMIN_PASS="q7Rf9Xk2Lm3pW8zBA" \
bash scripts/setup-ggid-erp.sh
```

### 2. Build and Run

```bash
cd examples/erp-demo

# Set GGID URL (optional, defaults to https://ggid.iot2.win)
export GGID_URL=https://ggid.iot2.win

# Run
mvn spring-boot:run
```

The app starts on **http://localhost:8090**.

### 3. Login and Test

Open http://localhost:8090 and login with:

| Username | Password | Can Access |
|----------|----------|------------|
| `sales_manager` | `ErpDemo2024!` | Orders, Inventory (read), Reports |
| `warehouse_manager` | `ErpDemo2024!` | Inventory (full), Orders (read/write) |
| `finance_officer` | `ErpDemo2024!` | Invoices, Payments (not visible to others) |
| `hr_manager` | `ErpDemo2024!` | Employees only |
| `production_manager` | `ErpDemo2024!` | Production, Inventory (read) |
| `erp_admin` | `ErpDemo2024!` | Everything |

### 4. Verify Fine-Grained Control

1. Login as `sales_manager` → go to Orders → you can **Approve** pending orders
2. Login as `warehouse_manager` → go to Orders → you can **Ship** but NOT **Approve**
3. Login as `finance_officer` → you see Invoices/Payments but NOT Inventory
4. Try to access a restricted URL directly (e.g. `/inventory/new` as `hr_manager`) → get 403

## Key Integration Points

### 1. JWT Verification (JwtVerifier.java)
```java
JwtVerifier verifier = new JwtVerifier("https://ggid.iot2.win/oauth/jwks");
GGIDUser user = verifier.verifyUser(token);
// user.userId, user.roles, user.scopes available
```

### 2. Permission Check (GGIDClient.java)
```java
PolicyResult result = ggidClient.checkPermission(token, userId, "inventory", "delete");
if (!result.isAllowed()) {
    throw new ForbiddenException(result.getReason());
}
```

### 3. Annotation-Based Guard (PermissionAspect.java)
```java
@RequirePermission(resource = "orders", action = "approve")
public String approveOrder(...) { ... }
```

## File Structure

```
examples/erp-demo/
├── pom.xml
├── README.md
├── scripts/
│   └── setup-ggid-erp.sh         # GGID configuration script
└── src/main/
    ├── java/com/example/erp/
    │   ├── ErpDemoApplication.java
    │   ├── config/
    │   │   ├── GgidConfig.java     # GGID client beans
    │   │   └── WebConfig.java      # Interceptor registration
    │   ├── controller/
    │   │   ├── ErpController.java  # Main ERP endpoints
    │   │   └── LoginController.java
    │   ├── model/
    │   │   ├── Product.java
    │   │   ├── Order.java
    │   │   └── ApiResponse.java
    │   ├── security/
    │   │   ├── AuthInterceptor.java     # JWT auth filter
    │   │   ├── PermissionAspect.java    # @RequirePermission AOP
    │   │   ├── RequirePermission.java   # Annotation
    │   │   └── ErpPermissions.java      # Local fallback matrix
    │   └── service/
    │       └── AuthService.java
    └── resources/
        ├── application.yml
        └── templates/               # Thymeleaf UI
            ├── login.html
            ├── dashboard.html
            ├── inventory.html
            ├── orders.html
            ├── reports.html
            └── my-permissions.html
```

## Answer: Can GGID Handle This?

**Yes.** GGID's RBAC system supports arbitrary `resource_type` + `action` combinations.

- Create custom permissions: `inventory:read`, `orders:approve`, `payments:write`
- Assign them to roles: `erp:sales_manager`, `erp:finance_officer`
- The Policy Decision Point (`POST /api/v1/policies/check`) evaluates in real-time
- Supports role inheritance, ABAC conditions, and wildcard actions

The ERP app defines its own module vocabulary (inventory, orders, invoices...) and
action vocabulary (read, write, delete, approve). GGID stores and enforces these
as first-class permissions — no different from built-in ones like `users:read`.
