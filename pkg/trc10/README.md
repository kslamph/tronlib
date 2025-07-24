# TRC10 Package

The TRC10 package provides comprehensive functionality for managing TRC10 tokens on the TRON blockchain.

## Features

### Asset Issue Management
- **CreateAssetIssue2**: Create new TRC10 tokens with comprehensive configuration options
- **GetAssetIssueByName**: Retrieve asset information by name
- **GetAssetIssueListByName**: Get list of assets matching a name
- **GetAssetIssueById**: Retrieve asset information by ID
- **GetPaginatedAssetIssueList**: Get paginated list of all assets

### Token Transfer Operations
- **TransferAsset2**: Transfer TRC10 tokens between addresses
- **ParticipateAssetIssue2**: Participate in token ICO/sale

### Frozen Asset Management
- **UnfreezeAsset2**: Unfreeze previously frozen assets

## Usage Example

```go
package main

import (
    "context"
    "log"
    
    "github.com/kslamph/tronlib/pkg/client"
    "github.com/kslamph/tronlib/pkg/trc10"
)

func main() {
    // Initialize client
    client := client.NewClient("grpc://127.0.0.1:50051")
    
    // Create TRC10 manager
    manager := trc10.NewManager(client)
    
    // Create a new asset
    frozenSupply := []trc10.FrozenSupply{
        {FrozenAmount: 1000000, FrozenDays: 30},
    }
    
    tx, err := manager.CreateAssetIssue2(
        context.Background(),
        "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", // owner
        "MyToken",                              // name
        "MTK",                                  // abbreviation
        1000000000,                             // total supply
        1,                                      // TRX num
        1,                                      // ICO num
        1640995200000,                          // start time
        1640995300000,                          // end time
        "My custom token",                      // description
        "https://mytoken.com",                  // URL
        1000,                                   // free asset net limit
        1000,                                   // public free asset net limit
        frozenSupply,                           // frozen supply
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Transfer assets
    transferTx, err := manager.TransferAsset2(
        context.Background(),
        "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", // from
        "TKx9RQveWvAcPTisx6QzSgYPVUjbmCCjpJ", // to
        "MyToken",                              // asset name
        1000,                                   // amount
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

## Validation

All functions include comprehensive input validation:
- Address format validation using the utils package
- Amount validation (positive values)
- Required field validation
- Time range validation for asset creation
- Frozen supply validation

## Error Handling

The package provides detailed error messages for:
- Invalid addresses
- Invalid amounts
- Missing required fields
- Network communication errors
- Transaction creation failures

## Design Patterns

The TRC10 package follows the established patterns from other packages:
- One gRPC call per function
- Proper input validation
- Clean separation between transaction creation and signing
- Consistent error handling
- Integration with the client and utils packages