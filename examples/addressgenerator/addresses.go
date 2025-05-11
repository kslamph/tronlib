package main

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"        // For file creation
	"os/signal" // For graceful shutdown
	"runtime"   // For CPU profiling
	"sync"      // For WaitGroup
	"syscall"   // For OS signals

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
)

const ethAddrLen = 20
const tronPrefixLen = 1
const checksumLen = 4
const totalAddressBytesLen = tronPrefixLen + ethAddrLen + checksumLen

func main() {
	// Open/Create a log file for found addresses
	logFile, err := os.OpenFile("found_addresses.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	fileLogger := log.New(logFile, "", log.LstdFlags|log.Ldate|log.Ltime)

	// Setup CPU profiling
	// f, err := os.Create("cpu.pprof")
	// if err != nil {
	// 	log.Fatal("could not create CPU profile: ", err)
	// }
	// defer f.Close()
	// if err := pprof.StartCPUProfile(f); err != nil {
	// 	log.Fatal("could not start CPU profile: ", err)
	// }
	// defer pprof.StopCPUProfile()

	// Start a goroutine to print the count every minute
	log.Println("Starting address generation...")

	var wg sync.WaitGroup
	done := make(chan struct{}) // Channel to signal goroutines to stop

	numWorkers := runtime.NumCPU() * 2
	log.Printf("Starting %d worker goroutines...\n", numWorkers)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			// log.Printf("Worker %d started\n", workerID)
			for {
				select {
				case <-done:
					// log.Printf("Worker %d stopping...\n", workerID)
					return
				default:
					address, key := newAccountOptimized()
					if IsMaxCharOptimized(address) || IsSameChar(address, 6) {
						log.Printf("Address: %s, Private Key: %s\n", address, key)
						fileLogger.Printf("Address: %s, Private Key: %s\n", address, key)
					}
				}
			}
		}(i)
	}

	// Graceful shutdown handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Printf("Received signal: %s. Shutting down...\n", sig)
		close(done) // Signal workers to stop
	}()

	log.Println("Application started. Press Ctrl+C to exit.")
	wg.Wait() // Wait for all worker goroutines to finish

	log.Println("All workers stopped. Finalizing profile.")
}

func newAccount() (string, string) {
	privKey, _ := crypto.GenerateKey()
	pubKey := privKey.PublicKey
	ethAddr := crypto.PubkeyToAddress(pubKey)

	// Add TRON prefix (0x41)
	tronBytes := append([]byte{0x41}, ethAddr.Bytes()...)

	h1 := sha256.Sum256(tronBytes)
	h2 := sha256.Sum256(h1[:])
	// checksum := h2[:4]...

	privateKeyBytes := crypto.FromECDSA(privKey)

	// Encode to base58
	return base58.Encode(append(tronBytes, h2[:4]...)), hex.EncodeToString(privateKeyBytes)
}
func newAccountOptimized() (string, string) {

	privKey, _ := crypto.GenerateKey() // This is a major part of the execution time
	pubKey := privKey.PublicKey
	ethAddr := crypto.PubkeyToAddress(pubKey)
	ethAddrBytes := ethAddr.Bytes() // This will always be ethAddrLen (20 bytes)

	// Pre-allocate slice for tronBytes + checksum to reduce appends
	addressBytes := make([]byte, totalAddressBytesLen)

	// Add TRON prefix (0x41)
	addressBytes[0] = 0x41
	copy(addressBytes[tronPrefixLen:], ethAddrBytes)

	// Calculate checksum directly on the relevant part of addressBytes
	// Hash prefix + ethAddr
	h1 := sha256.Sum256(addressBytes[:tronPrefixLen+ethAddrLen])
	h2 := sha256.Sum256(h1[:])
	// Append checksum
	copy(addressBytes[tronPrefixLen+ethAddrLen:], h2[:checksumLen])

	privateKeyBytes := crypto.FromECDSA(privKey) // Another significant part

	// Encode to base58
	return base58.Encode(addressBytes), hex.EncodeToString(privateKeyBytes)
}

func IsSameChar(s string, length int) bool {
	if len(s) < length {
		return false
	}
	if length == 0 {
		return true
	}
	substr := s[len(s)-length:]
	character := substr[0]
	// check if the last len characters are the same
	for i := 1; i < length; i++ {
		if substr[i] != character {
			return false
		}
	}
	return true
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// IsMaxChar checks if a string of fixed length 34 (standard Tron address) ends with
// "Max" or "max" followed by 2 or 3 identical digits.
func IsMaxChar(s string) bool {
	const expectedLen = 34
	if len(s) != expectedLen {
		// log.Printf("Warning: IsMaxChar received string of length %d, expected %d: %s", len(s), expectedLen, s)
		return false // Prevent panic on unexpected length
	}

	// Check for "MaxDDD" or "maxDDD" (total 6 characters: 3 for "Max", 3 for "DDD")
	// String: ... | M | a | x | D | D | D |
	// Index:  ... |28 |29 |30 |31 |32 |33 | (0-based for a 34-char string)
	// "Max"/"max" part: s[28:31]
	// Digits part: s[31], s[32], s[33]
	key3Part := s[expectedLen-6 : expectedLen-3] // s[28:31]
	if key3Part == "Max" || key3Part == "max" {
		d1 := s[expectedLen-3] // s[31]
		d2 := s[expectedLen-2] // s[32]
		d3 := s[expectedLen-1] // s[33]
		if isDigit(d1) && d1 == d2 && d2 == d3 {
			// No need to check isDigit(d2) and isDigit(d3) explicitly
			// because if d1 is a digit, and d2 and d3 are equal to d1,
			// they must also be digits.
			return true
		}
	}

	// Check for "MaxDD" or "maxDD" (total 5 characters: 3 for "Max", 2 for "DD")
	// String: ... | M | a | x | D | D |
	// Index:  ... |29 |30 |31 |32 |33 | (0-based for a 34-char string)
	// "Max"/"max" part: s[29:32]
	// Digits part: s[32], s[33]
	key2Part := s[expectedLen-5 : expectedLen-2] // s[29:32]
	if key2Part == "Max" || key2Part == "max" {
		d1 := s[expectedLen-2] // s[32]
		d2 := s[expectedLen-1] // s[33]
		if isDigit(d1) && d1 == d2 {
			// Similar to above, isDigit(d2) is implied.
			return true
		}
	}

	return false
}
func IsMaxCharOptimized(s string) bool {

	// Check for "MaxDD" or "maxDD" (ends with 2 identical digits)
	// Indices for a 34-char string:
	// M/m  a   x   D1  D2
	// s[29]s[30]s[31]s[32]s[33]

	charD1_2 := s[32] // s[32]
	charD2_2 := s[33] // s[33]

	if isDigit(charD1_2) && charD1_2 == charD2_2 {
		// Now check for "Max" or "max" preceding these digits
		if (s[29] == 'M' || s[29] == 'm') && // s[29]
			s[30] == 'a' && // s[30]
			s[31] == 'x' { // s[31]
			return true
		}
	}

	// Check for "MaxDDD" or "maxDDD" (ends with 3 identical digits)
	// Indices for a 34-char string:
	// M/m  a   x   D1  D2  D3
	// s[28]s[29]s[30]s[31]s[32]s[33]

	charD1_3 := s[31] // s[31]
	charD2_3 := s[32]
	charD3_3 := s[33]

	if isDigit(charD1_3) && charD1_3 == charD2_3 && charD2_3 == charD3_3 {
		// Now check for "Max" or "max" preceding these digits
		if (s[28] == 'M' || s[28] == 'm') && // s[28]
			s[29] == 'a' && // s[29]
			s[30] == 'x' { // s[30]
			return true
		}
	}

	return false
}
