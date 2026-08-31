// Package tron is the v2 vocabulary package: the shared types every other
// v2 package speaks. Address is a TRON address as a value type; SUN is the
// integer token unit; Error and Code are the machine-parseable error model
// every exported v2 function returns.
//
// The package has zero internal dependencies and performs no I/O: no ctx,
// no network, no goroutines — it only defines what the other packages mean.
package tron
