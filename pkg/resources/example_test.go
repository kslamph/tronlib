package resources_test

import (
	"context"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/resources"
	"github.com/kslamph/tronlib/pkg/types"
)

// ExampleNewManager demonstrates freezing energy.
func ExampleNewManager() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
	defer cli.Close()

	rm := resources.NewManager(cli)
	owner, _ := types.NewAddress("Townerxxxxxxxxxxxxxxxxxxxxxxxxxx")
	_, _ = rm.FreezeBalanceV2(ctx, owner, 1_000_000, resources.ResourceTypeEnergy)
}
