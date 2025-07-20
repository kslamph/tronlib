package main

import (
	"context"
	"fmt"
	"log"
	"time"

	// Import the generated gRPC code. Update this with your module name.
	api "github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// The address of the gRPC server.
	serverAddr = "localhost:50051"
)

func main() {
	// 1. Establish a connection to the server.
	// We're using `insecure.NewCredentials()` because this is a local example
	// without TLS. For production, you'd use proper credentials.
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	// Defer closing the connection until the function returns.
	defer conn.Close()

	// 2. Create a client stub for the Greeter service.
	// This `c` object is what you'll use to make RPC calls.

	walletClient := api.NewWalletClient(conn)

	// 3. Prepare the RPC call.
	// We create a context with a timeout to prevent the client from
	// waiting indefinitely if the server is unresponsive.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// The name we want to send in our request.
	address := "TXJgMdjVX5dKiQaUi9QobwNxtSQaFqccvd"
	addressBytes := types.MustNewAddress(address).Bytes()

	result, err := walletClient.GetContract(ctx, &api.BytesMessage{
		Value: addressBytes,
	})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

	// 5. Print the server's response.
	fmt.Printf("✅ Server responded: %v\n", result)
}

/*
```

### Explanation of the Code

1.  **`grpc.NewClient(...)`**: This is the core function that creates a connection to the gRPC server at `serverAddr`.
    *   `grpc.WithTransportCredentials(insecure.NewCredentials())` is used here for simplicity. It disables TLS encryption. **Never use this in production.** For a real-world application, you would configure TLS credentials here.
2.  **`defer conn.Close()`**: This is idiomatic Go. It ensures that the connection is properly closed when the `main` function finishes, releasing the resources.
3.  **`pb.NewGreeterClient(conn)`**: This function comes from the generated `greeter_grpc.pb.go` file. It takes the active connection (`conn`) and returns a `GreeterClient` instance, which is the client stub. This stub has methods that correspond to the RPCs defined in your `.proto` file (e.g., `SayHello`).
4.  **`context.WithTimeout(...)`**: All gRPC calls in Go should be made with a `context`. This allows you to control deadlines, timeouts, and cancellation. Here, we're saying the call should fail if it takes longer than one second.
5.  **`c.SayHello(ctx, ...)`**: This is the actual remote procedure call.
    *   The first argument is the `context`.
    *   The second argument is a pointer to the request message struct (`&pb.HelloRequest{...}`).
    *   It returns the response message (`HelloReply`) and an `error`.
6.  **`r.GetMessage()`**: The generated Go structs for your protobuf messages include helper methods like `Get...()` to access their fields safely.

### How to Run It

1.  Make sure your gRPC server is running.
2.  Update the `go_package` option in your `.proto` file and the `import` path in `client/main.go` to match your project's Go module name (from your `go.mod` file).
3.  Open a new terminal in your project root.
4.  Run `go mod tidy` to fetch the necessary dependencies.
5.  Run the client:
    ```bash
    go run client/main.go
    ```
*/
