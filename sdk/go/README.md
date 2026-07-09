# ggid-go

GGID Go SDK — integrate GGID IAM into your Go backend.

## Installation

```bash
go get github.com/ggid/ggid/sdk/go
```

## Usage

```go
import ggid "github.com/ggid/ggid/sdk/go"

client := ggid.New("https://iam.example.com", ggid.WithAPIKey("your-api-key"))

// Verify a JWT access token
userInfo, err := client.VerifyToken(ctx, accessToken)

// Check permission
allowed, err := client.CheckPermission(ctx, userID, "iam:users", "read")
```
