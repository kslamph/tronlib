package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"hash/crc32"
	"math/big"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const defaultTimeout = 30 * time.Second

// GCPKMSClient implements KMSClientInterface using Google Cloud KMS
type GCPKMSClient struct {
	client     *kms.KeyManagementClient
	projectID  string
	locationID string
	keyRingID  string
}

// NewGCPKMSClient creates a new Google Cloud KMS client
func NewGCPKMSClient(projectID, locationID, keyRingID string) (*GCPKMSClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS client: %w", err)
	}

	return &GCPKMSClient{
		client:     client,
		projectID:  projectID,
		locationID: locationID,
		keyRingID:  keyRingID,
	}, nil
}

// recoverRS extracts R and S values from an ASN.1 signature
func recoverRS(derSignature []byte) (r, s *big.Int, err error) {
	var sig struct {
		R, S *big.Int
	}
	if _, err := asn1.Unmarshal(derSignature, &sig); err != nil {
		return nil, nil, fmt.Errorf("failed to parse ASN.1 signature: %w", err)
	}
	return sig.R, sig.S, nil
}

// SignDigest signs a digest using the specified key version
func (g *GCPKMSClient) SignDigest(keyName string, digest []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	keyPath := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s/cryptoKeyVersions/1", g.projectID, g.locationID, g.keyRingID, keyName)
	digestCRC32C := crc32.Checksum(digest, crc32.MakeTable(crc32.Castagnoli))
	req := &kmspb.AsymmetricSignRequest{
		Name: keyPath,
		Digest: &kmspb.Digest{
			Digest: &kmspb.Digest_Sha256{
				Sha256: digest,
			},
		},
		DigestCrc32C: wrapperspb.Int64(int64(digestCRC32C)),
	}

	var resultErr error
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		result, err := g.client.AsymmetricSign(ctx, req)
		if err != nil {
			resultErr = err
			continue
		}

		if !result.VerifiedDigestCrc32C {
			resultErr = fmt.Errorf("request corrupted in-transit")
			continue
		}

		if result.SignatureCrc32C != nil {
			sigCRC32C := crc32.Checksum(result.Signature, crc32.MakeTable(crc32.Castagnoli))
			if int64(sigCRC32C) != result.SignatureCrc32C.Value {
				resultErr = fmt.Errorf("response corrupted in-transit")
				continue
			}
		}

		// Convert ASN.1 DER signature to R || S || V format
		r, s, err := recoverRS(result.Signature)
		if err != nil {
			resultErr = err
			continue
		}

		sig := make([]byte, 65)
		r.FillBytes(sig[:32])
		s.FillBytes(sig[32:64])
		sig[64] = 0 // V will be adjusted by KMSAccount

		return sig, nil
	}

	return nil, fmt.Errorf("failed to sign after %d attempts: %w", maxRetries, resultErr)
}

// CreateKey creates a new asymmetric key for signing in Google Cloud KMS
func (g *GCPKMSClient) CreateKey() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Create key ring if it doesn't exist
	keyRingName := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s", g.projectID, g.locationID, g.keyRingID)
	keyRingReq := &kmspb.CreateKeyRingRequest{
		Parent:    fmt.Sprintf("projects/%s/locations/%s", g.projectID, g.locationID),
		KeyRingId: g.keyRingID,
	}
	_, err := g.client.CreateKeyRing(ctx, keyRingReq)
	if err != nil {
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.AlreadyExists {
			return "", fmt.Errorf("failed to create key ring: %w", err)
		}
	}

	// Create a new crypto key with ECDSA P256K1
	keyID := fmt.Sprintf("tron-signing-key-%d", time.Now().Unix())
	createCryptoKeyReq := &kmspb.CreateCryptoKeyRequest{
		Parent:      keyRingName,
		CryptoKeyId: keyID,
		CryptoKey: &kmspb.CryptoKey{
			Purpose: kmspb.CryptoKey_ASYMMETRIC_SIGN,
			VersionTemplate: &kmspb.CryptoKeyVersionTemplate{
				Algorithm: kmspb.CryptoKeyVersion_EC_SIGN_SECP256K1_SHA256,
			},
		},
	}

	key, err := g.client.CreateCryptoKey(ctx, createCryptoKeyReq)
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			// If key exists, get its name
			getReq := &kmspb.GetCryptoKeyRequest{Name: fmt.Sprintf("%s/cryptoKeys/%s", keyRingName, keyID)}
			key, err = g.client.GetCryptoKey(ctx, getReq)
			if err != nil {
				return "", fmt.Errorf("failed to get existing crypto key: %w", err)
			}
		} else {
			return "", fmt.Errorf("failed to create crypto key: %w", err)
		}
	}

	return key.Name, nil
}

// GetPublicKey retrieves the public key for the given key version
func (g *GCPKMSClient) GetPublicKey(keyName string) (*ecdsa.PublicKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	keyPath := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s/cryptoKeyVersions/1", g.projectID, g.locationID, g.keyRingID, keyName)
	response, err := g.client.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{
		Name: keyPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	block, rest := pem.Decode([]byte(response.GetPem()))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	if len(rest) > 0 {
		return nil, fmt.Errorf("unexpected data after PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not ECDSA")
	}

	return ecdsaPub, nil
}

// Close closes the KMS client
func (g *GCPKMSClient) Close() error {
	return g.client.Close()
}
