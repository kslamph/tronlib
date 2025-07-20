.PHONY: build clean run-collector run-query help

# Build all binaries
build:
	@echo "Building event signature collector..."
	go build -o event-collector cmd/collector/main.go
	@echo "Building query tool..."
	go build -o query cmd/query/main.go
	@echo "Building main collector..."
	go build -o main main.go
	@echo "Build complete!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f event-collector query main
	@echo "Clean complete!"

# Run the collector with default settings
run-collector:
	@echo "Starting event signature collector..."
	./event-collector

# Run the collector with custom settings
run-collector-custom:
	@echo "Starting event signature collector with custom settings..."
	./event-collector -node 127.0.0.1:50051 -db event_signatures.db -interval 2s

# Show help for query tool
query-help:
	@echo "Query tool help:"
	./query -help

# Show statistics
stats:
	@echo "Event signature statistics:"
	./query -stats

# Show recent signatures
recent:
	@echo "Recent event signatures:"
	./query -limit 10

# Show most used signatures
popular:
	@echo "Most used event signatures:"
	./query -sort usage_count -order desc -limit 10

# Clean up unknown events
cleanup:
	@echo "Cleaning up unknown events..."
	./query -cleanup

# Export all signatures to JSON
export:
	@echo "Exporting signatures to signatures.json..."
	./query -export signatures.json

# Show help
help:
	@echo "Event Signature Collector - Available commands:"
	@echo ""
	@echo "Build commands:"
	@echo "  make build              - Build all binaries"
	@echo "  make clean              - Clean build artifacts"
	@echo ""
	@echo "Collector commands:"
	@echo "  make run-collector      - Run collector with default settings"
	@echo "  make run-collector-custom - Run collector with custom settings"
	@echo ""
	@echo "Query commands:"
	@echo "  make query-help         - Show query tool help"
	@echo "  make stats              - Show statistics"
	@echo "  make recent             - Show recent signatures"
	@echo "  make popular            - Show most used signatures"
	@echo "  make export             - Export to JSON"
	@echo ""
	@echo "Manual usage:"
	@echo "  ./event-collector [options]  - Run collector"
	@echo "  ./query [options]            - Query signatures"
	@echo "  ./main                       - Run simple collector" 