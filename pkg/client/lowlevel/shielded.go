// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// Package lowlevel provides 1:1 wrappers around WalletClient gRPC methods.
package lowlevel

import (
	"context"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// Shielded transaction related gRPC calls

// CreateShieldedTransaction creates a shielded transaction
func CreateShieldedTransaction(cp ConnProvider, ctx context.Context, req *api.PrivateParameters) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "create shielded transaction", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.CreateShieldedTransaction(ctx, req)
	})
}

// GetMerkleTreeVoucherInfo gets merkle tree voucher info
func GetMerkleTreeVoucherInfo(cp ConnProvider, ctx context.Context, req *core.OutputPointInfo) (*core.IncrementalMerkleVoucherInfo, error) {
	return Call(cp, ctx, "get merkle tree voucher info", func(client api.WalletClient, ctx context.Context) (*core.IncrementalMerkleVoucherInfo, error) {
		return client.GetMerkleTreeVoucherInfo(ctx, req)
	})
}

// ScanNoteByIvk scans notes by ivk
func ScanNoteByIvk(cp ConnProvider, ctx context.Context, req *api.IvkDecryptParameters) (*api.DecryptNotes, error) {
	return Call(cp, ctx, "scan note by ivk", func(client api.WalletClient, ctx context.Context) (*api.DecryptNotes, error) {
		return client.ScanNoteByIvk(ctx, req)
	})
}

// ScanAndMarkNoteByIvk scans and marks notes by ivk
func ScanAndMarkNoteByIvk(cp ConnProvider, ctx context.Context, req *api.IvkDecryptAndMarkParameters) (*api.DecryptNotesMarked, error) {
	return Call(cp, ctx, "scan and mark note by ivk", func(client api.WalletClient, ctx context.Context) (*api.DecryptNotesMarked, error) {
		return client.ScanAndMarkNoteByIvk(ctx, req)
	})
}

// ScanNoteByOvk scans notes by ovk
func ScanNoteByOvk(cp ConnProvider, ctx context.Context, req *api.OvkDecryptParameters) (*api.DecryptNotes, error) {
	return Call(cp, ctx, "scan note by ovk", func(client api.WalletClient, ctx context.Context) (*api.DecryptNotes, error) {
		return client.ScanNoteByOvk(ctx, req)
	})
}

// GetSpendingKey gets spending key
func GetSpendingKey(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.BytesMessage, error) {
	return Call(cp, ctx, "get spending key", func(client api.WalletClient, ctx context.Context) (*api.BytesMessage, error) {
		return client.GetSpendingKey(ctx, req)
	})
}

// GetExpandedSpendingKey gets expanded spending key
func GetExpandedSpendingKey(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*api.ExpandedSpendingKeyMessage, error) {
	return Call(cp, ctx, "get expanded spending key", func(client api.WalletClient, ctx context.Context) (*api.ExpandedSpendingKeyMessage, error) {
		return client.GetExpandedSpendingKey(ctx, req)
	})
}

// GetAkFromAsk gets ak from ask
func GetAkFromAsk(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*api.BytesMessage, error) {
	return Call(cp, ctx, "get ak from ask", func(client api.WalletClient, ctx context.Context) (*api.BytesMessage, error) {
		return client.GetAkFromAsk(ctx, req)
	})
}

// GetNkFromNsk gets nk from nsk
func GetNkFromNsk(cp ConnProvider, ctx context.Context, req *api.BytesMessage) (*api.BytesMessage, error) {
	return Call(cp, ctx, "get nk from nsk", func(client api.WalletClient, ctx context.Context) (*api.BytesMessage, error) {
		return client.GetNkFromNsk(ctx, req)
	})
}

// GetIncomingViewingKey gets incoming viewing key
func GetIncomingViewingKey(cp ConnProvider, ctx context.Context, req *api.ViewingKeyMessage) (*api.IncomingViewingKeyMessage, error) {
	return Call(cp, ctx, "get incoming viewing key", func(client api.WalletClient, ctx context.Context) (*api.IncomingViewingKeyMessage, error) {
		return client.GetIncomingViewingKey(ctx, req)
	})
}

// GetDiversifier gets diversifier
func GetDiversifier(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.DiversifierMessage, error) {
	return Call(cp, ctx, "get diversifier", func(client api.WalletClient, ctx context.Context) (*api.DiversifierMessage, error) {
		return client.GetDiversifier(ctx, req)
	})
}

// GetNewShieldedAddress generates a new shielded address
func GetNewShieldedAddress(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.ShieldedAddressInfo, error) {
	return Call(cp, ctx, "get new shielded address", func(client api.WalletClient, ctx context.Context) (*api.ShieldedAddressInfo, error) {
		return client.GetNewShieldedAddress(ctx, req)
	})
}

// GetZenPaymentAddress gets zen payment address
func GetZenPaymentAddress(cp ConnProvider, ctx context.Context, req *api.IncomingViewingKeyDiversifierMessage) (*api.PaymentAddressMessage, error) {
	return Call(cp, ctx, "get zen payment address", func(client api.WalletClient, ctx context.Context) (*api.PaymentAddressMessage, error) {
		return client.GetZenPaymentAddress(ctx, req)
	})
}

// GetRcm gets rcm
func GetRcm(cp ConnProvider, ctx context.Context, req *api.EmptyMessage) (*api.BytesMessage, error) {
	return Call(cp, ctx, "get rcm", func(client api.WalletClient, ctx context.Context) (*api.BytesMessage, error) {
		return client.GetRcm(ctx, req)
	})
}

// IsSpend checks if note is spent
func IsSpend(cp ConnProvider, ctx context.Context, req *api.NoteParameters) (*api.SpendResult, error) {
	return Call(cp, ctx, "is spend", func(client api.WalletClient, ctx context.Context) (*api.SpendResult, error) {
		return client.IsSpend(ctx, req)
	})
}

// CreateShieldedTransactionWithoutSpendAuthSig creates a shielded transaction without spend auth signature
func CreateShieldedTransactionWithoutSpendAuthSig(cp ConnProvider, ctx context.Context, req *api.PrivateParametersWithoutAsk) (*api.TransactionExtention, error) {
	return TxCall(cp, ctx, "create shielded transaction without spend auth sig", func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return client.CreateShieldedTransactionWithoutSpendAuthSig(ctx, req)
	})
}

// GetShieldTransactionHash gets the hash of a shielded transaction
func GetShieldTransactionHash(cp ConnProvider, ctx context.Context, req *core.Transaction) (*api.BytesMessage, error) {
	return Call(cp, ctx, "get shield transaction hash", func(client api.WalletClient, ctx context.Context) (*api.BytesMessage, error) {
		return client.GetShieldTransactionHash(ctx, req)
	})
}

// CreateSpendAuthSig creates spend auth signature
func CreateSpendAuthSig(cp ConnProvider, ctx context.Context, req *api.SpendAuthSigParameters) (*api.BytesMessage, error) {
	return Call(cp, ctx, "create spend auth sig", func(client api.WalletClient, ctx context.Context) (*api.BytesMessage, error) {
		return client.CreateSpendAuthSig(ctx, req)
	})
}

// CreateShieldNullifier creates a shield nullifier
func CreateShieldNullifier(cp ConnProvider, ctx context.Context, req *api.NfParameters) (*api.BytesMessage, error) {
	return Call(cp, ctx, "create shield nullifier", func(client api.WalletClient, ctx context.Context) (*api.BytesMessage, error) {
		return client.CreateShieldNullifier(ctx, req)
	})
}

// CreateShieldedContractParameters creates shielded contract parameters
func CreateShieldedContractParameters(cp ConnProvider, ctx context.Context, req *api.PrivateShieldedTRC20Parameters) (*api.ShieldedTRC20Parameters, error) {
	return Call(cp, ctx, "create shielded contract parameters", func(client api.WalletClient, ctx context.Context) (*api.ShieldedTRC20Parameters, error) {
		return client.CreateShieldedContractParameters(ctx, req)
	})
}

// CreateShieldedContractParametersWithoutAsk creates shielded contract parameters without ask
func CreateShieldedContractParametersWithoutAsk(cp ConnProvider, ctx context.Context, req *api.PrivateShieldedTRC20ParametersWithoutAsk) (*api.ShieldedTRC20Parameters, error) {
	return Call(cp, ctx, "create shielded contract parameters without ask", func(client api.WalletClient, ctx context.Context) (*api.ShieldedTRC20Parameters, error) {
		return client.CreateShieldedContractParametersWithoutAsk(ctx, req)
	})
}

// ScanShieldedTRC20NotesByIvk scans shielded TRC20 notes by ivk
func ScanShieldedTRC20NotesByIvk(cp ConnProvider, ctx context.Context, req *api.IvkDecryptTRC20Parameters) (*api.DecryptNotesTRC20, error) {
	return Call(cp, ctx, "scan shielded trc20 notes by ivk", func(client api.WalletClient, ctx context.Context) (*api.DecryptNotesTRC20, error) {
		return client.ScanShieldedTRC20NotesByIvk(ctx, req)
	})
}

// ScanShieldedTRC20NotesByOvk scans shielded TRC20 notes by ovk
func ScanShieldedTRC20NotesByOvk(cp ConnProvider, ctx context.Context, req *api.OvkDecryptTRC20Parameters) (*api.DecryptNotesTRC20, error) {
	return Call(cp, ctx, "scan shielded trc20 notes by ovk", func(client api.WalletClient, ctx context.Context) (*api.DecryptNotesTRC20, error) {
		return client.ScanShieldedTRC20NotesByOvk(ctx, req)
	})
}

// IsShieldedTRC20ContractNoteSpent checks if a shielded TRC20 contract note is spent
func IsShieldedTRC20ContractNoteSpent(cp ConnProvider, ctx context.Context, req *api.NfTRC20Parameters) (*api.NullifierResult, error) {
	return Call(cp, ctx, "is shielded trc20 contract note spent", func(client api.WalletClient, ctx context.Context) (*api.NullifierResult, error) {
		return client.IsShieldedTRC20ContractNoteSpent(ctx, req)
	})
}

// GetTriggerInputForShieldedTRC20Contract gets trigger input for shielded TRC20 contract
func GetTriggerInputForShieldedTRC20Contract(cp ConnProvider, ctx context.Context, req *api.ShieldedTRC20TriggerContractParameters) (*api.BytesMessage, error) {
	return Call(cp, ctx, "get trigger input for shielded trc20 contract", func(client api.WalletClient, ctx context.Context) (*api.BytesMessage, error) {
		return client.GetTriggerInputForShieldedTRC20Contract(ctx, req)
	})
}
