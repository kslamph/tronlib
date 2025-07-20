#!/bin/bash

# Exit on error
set -e

# Directory containing this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Root directory of the project (parent of scripts directory)
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Input and output directories
PROTO_ROOT="$PROJECT_ROOT/protos"
OUTPUT_DIR="$PROJECT_ROOT/pb"

# Create minimal googleapis directory with only needed files
MINIMAL_GOOGLEAPIS="$PROJECT_ROOT/tmp/googleapis"
mkdir -p "$MINIMAL_GOOGLEAPIS/google/api"

# Download only the annotations.proto file if it doesn't exist
if [ ! -f "$MINIMAL_GOOGLEAPIS/google/api/annotations.proto" ]; then
    echo "Downloading minimal Google APIs (annotations.proto)..."
    curl -s -o "$MINIMAL_GOOGLEAPIS/google/api/annotations.proto" \
        "https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto"
    
    # Also download http.proto as it's a dependency of annotations.proto
    curl -s -o "$MINIMAL_GOOGLEAPIS/google/api/http.proto" \
        "https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto"
    
    echo "Minimal Google APIs downloaded successfully!"
else
    echo "Minimal Google APIs already exists, skipping download."
fi

rm -rf "$OUTPUT_DIR"
# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Generate M options for core files
M_OPTIONS=""
M_GRPC_OPTIONS=""

# Find all core proto files and generate M options
find "$PROTO_ROOT/core" -type f -name "*.proto" ! -empty ! -path "*/core/tron/*" | while read proto_file; do
    rel_path=${proto_file#$PROTO_ROOT/}
    # Map both the full path and the base name to handle different import styles
    M_OPTIONS="$M_OPTIONS --go_opt=M$rel_path=github.com/kslamph/tronlib/pb/core"
    M_GRPC_OPTIONS="$M_GRPC_OPTIONS --go-grpc_opt=M$rel_path=github.com/kslamph/tronlib/pb/core"
done

# Find all proto files recursively, excluding empty files and the tron directory
find "$PROTO_ROOT" -type f -name "*.proto" ! -empty ! -path "*/core/tron/*" | while read proto_file; do
    # Get the relative path from PROTO_ROOT
    rel_path=${proto_file#$PROTO_ROOT/}
    # Get the directory part of the relative path
    dir_path=$(dirname "$rel_path")

    # For core files, we want them directly in pb/core
    if [[ "$dir_path" == core/* ]]; then
        target_dir="core"
        output_path="$OUTPUT_DIR/core/$(basename "$proto_file" .proto)"
    else
        target_dir="$dir_path"
        output_path="$OUTPUT_DIR/$dir_path/$(basename "$proto_file" .proto)"
    fi

    # Create the output directory structure
    mkdir -p "$(dirname "$output_path")"

    echo "Generating Go code for: $rel_path"

    # Run protoc compiler
    protoc \
        -I "$PROTO_ROOT" \
        -I "$MINIMAL_GOOGLEAPIS" \
        -I "/usr/include" \
        --go_out="$OUTPUT_DIR" \
        --go_opt=paths=source_relative \
        --go_opt=Mgithub.com/tronprotocol/grpc-gateway/core/Tron.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mgithub.com/tronprotocol/grpc-gateway/core/Discover.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mgithub.com/tronprotocol/grpc-gateway/core/TronInventoryItems.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/account_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/asset_issue_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/balance_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/common.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/exchange_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/market_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/proposal_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/shield_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/smart_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/storage_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/vote_asset_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/witness_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mcore/Tron.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mcore/Discover.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mcore/TronInventoryItems.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mcore/contract/account_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mcore/contract/asset_issue_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mcore/contract/balance_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mcore/contract/common.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mcore/contract/exchange_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mcore/contract/market_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mcore/contract/proposal_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mcore/contract/shield_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mcore/contract/smart_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mcore/contract/storage_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mcore/contract/vote_asset_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=Mcore/contract/witness_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go_opt=M${rel_path}=github.com/kslamph/tronlib/pb/${target_dir} \
        --go-grpc_out="$OUTPUT_DIR" \
        --go-grpc_opt=paths=source_relative \
        --go-grpc_opt=Mgithub.com/tronprotocol/grpc-gateway/core/Tron.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mgithub.com/tronprotocol/grpc-gateway/core/Discover.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mgithub.com/tronprotocol/grpc-gateway/core/TronInventoryItems.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/account_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/asset_issue_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/balance_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/common.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/exchange_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/market_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/proposal_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/shield_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/smart_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/storage_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/vote_asset_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mgithub.com/tronprotocol/grpc-gateway/core/contract/witness_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mcore/Tron.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mcore/Discover.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mcore/TronInventoryItems.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mcore/contract/account_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mcore/contract/asset_issue_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mcore/contract/balance_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mcore/contract/common.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mcore/contract/exchange_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mcore/contract/market_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mcore/contract/proposal_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mcore/contract/shield_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mcore/contract/smart_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mcore/contract/storage_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mcore/contract/vote_asset_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=Mcore/contract/witness_contract.proto=github.com/kslamph/tronlib/pb/core \
        --go-grpc_opt=M${rel_path}=github.com/kslamph/tronlib/pb/${target_dir} \
        "$proto_file"
done

# Move all core/contract/*.pb.go files to core/
if [ -d "$OUTPUT_DIR/core/contract" ]; then
    mv "$OUTPUT_DIR/core/contract/"*.pb.go "$OUTPUT_DIR/core/"
    rm -rf "$OUTPUT_DIR/core/contract"
fi

echo "Proto generation completed successfully!"
echo "Minimal Google APIs are stored in: $MINIMAL_GOOGLEAPIS"