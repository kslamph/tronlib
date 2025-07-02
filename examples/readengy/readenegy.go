package main

import (
	"fmt"
	"log"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

func main() {
	client, err := client.NewClient(client.ClientConfig{
		NodeAddress: "grpc.trongrid.io:50051",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	addr, err := types.NewAddress("TUAEpR16Th4pLbjECPVYTgvPyStNKyB54h")
	if err != nil {
		log.Fatalf("Failed to create receiver address: %v", err)
	}

	var lastused int64
	for {

		res, err := client.GetAccountResource(addr)
		if err != nil {
			log.Fatalf("Failed to get account resource: %v", err)
		}

		if res.GetEnergyUsed() < lastused || lastused == 0 {
			fmt.Printf("Energy restored: %d Energey Used: %d , Energy Limit: %d, Energy Available: %d\n", lastused-res.GetEnergyUsed(), res.GetEnergyUsed(), res.GetEnergyLimit(), res.GetEnergyLimit()-res.GetEnergyUsed())
			fmt.Printf("Ratio to Used: %.6f\n", (float64(lastused-res.GetEnergyUsed())*(20*60*24-8))/float64(res.GetEnergyUsed()))
			fmt.Printf("Ratio to Energy Limit: %.6f\n", (float64(lastused-res.GetEnergyUsed())*(20*60*24-8))/float64(res.GetEnergyLimit()))
		}

		lastused = res.GetEnergyUsed()
		time.Sleep(3 * time.Second)
	}
}
