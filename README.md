# lyhna-go

Go SDK for the Lyhna governance API. Every action gets a cryptographic receipt — APPROVED, REFUSED, or ESCALATED — before it executes.

## Install

```bash
go get github.com/lyhna-ai/lyhna-go
```

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "os"

    lyhna "github.com/lyhna-ai/lyhna-go"
)

func main() {
    client := lyhna.NewClient(os.Getenv("LYHNA_API_KEY"))

    receipt, err := client.Bind(context.Background(), lyhna.BindRequest{
        ActionType:    "deploy_production",
        Intent:        "release_v3",
        IntentVersion: "1.0",
        Payload:       map[string]interface{}{"env": "production"},
    })
    if err != nil {
        panic(err)
    }

    if receipt.Outcome == "APPROVED" {
        fmt.Println("Deploying:", receipt.ReceiptID)
    }
}
```

## Package-level convenience

```go
os.Setenv("LYHNA_API_KEY", "lyhna_...")

receipt, err := lyhna.Bind(context.Background(), lyhna.BindRequest{
    ActionType:    "deploy_production",
    Intent:        "release_v3",
    IntentVersion: "1.0",
})
```

## Verify a receipt

```go
result, err := client.VerifyReceipt(context.Background(), receipt)
if result.Valid {
    fmt.Println("Receipt verified")
}
```

## Options

```go
client := lyhna.NewClient("lyhna_...",
    lyhna.WithBaseURL("https://custom.lyhna.com"),
    lyhna.WithTimeout(30 * time.Second),
)
```

## Error handling

```go
receipt, err := client.Bind(ctx, req)
if err != nil {
    switch err.(type) {
    case *lyhna.AuthError:
        // 401 or 403
    case *lyhna.TimeoutError:
        // request timed out
    case *lyhna.LyhnaError:
        // other API error
    }
}
```

## Links

- [Documentation](https://docs.lyhna.com)
- [Dashboard](https://www.lyhna.com)
