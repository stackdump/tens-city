#!/bin/bash
# Example workflow demonstrating the tens-city CLI tools

set -e

echo "=== Tens City CLI Workflow Example ==="
echo ""

# Configuration
STORE_DIR="data"
EXAMPLE_FILE="examples/petrinet.jsonld"

echo "Step 1: Seal a JSON-LD file to filesystem"
echo "Command: ./seal -in $EXAMPLE_FILE -store $STORE_DIR"
./seal -in "$EXAMPLE_FILE" -store "$STORE_DIR"
echo ""

# Extract the CID from the output
CID=$(./seal -in "$EXAMPLE_FILE" -store "$STORE_DIR" 2>&1 | grep "sealed as" | awk '{print $3}')
echo "Generated CID: $CID"
echo ""

echo "Step 2: Start the web server"
echo "Command: ./webserver -addr :8080 -store $STORE_DIR -public public"
echo "Note: The server will serve sealed objects from the filesystem"
echo ""

echo "=== Workflow Complete ==="
echo ""
echo "To test the full workflow:"
echo "1. Run: ./webserver -addr :8080 -store $STORE_DIR -public public"
echo "2. Visit: http://localhost:8080/o/$CID"
echo "3. Or access via the web UI at: http://localhost:8080"

