#!/bin/bash
# Example workflow demonstrating the tens-city CLI tools

set -e

echo "=== Tens City CLI Workflow Example ==="
echo ""

# Configuration
DB_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
STORE_DIR="data"
EXAMPLE_FILE="examples/petrinet.jsonld"

echo "Step 1: Seal a JSON-LD file to filesystem"
echo "Command: ./seal -in $EXAMPLE_FILE -store $STORE_DIR"
./seal -in "$EXAMPLE_FILE" -store "$STORE_DIR"
echo ""

# Extract the CID from the output (this is just for demonstration)
CID=$(./seal -in "$EXAMPLE_FILE" -store "$STORE_DIR" 2>&1 | grep "sealed as" | awk '{print $3}')
echo "Generated CID: $CID"
echo ""

echo "Step 2: Import JSON-LD file directly to database"
echo "Command: ./edge import -db <DATABASE_URL> -file $EXAMPLE_FILE"
echo "Note: Make sure docker-compose is running (docker-compose up -d)"
echo "Note: Run migrations first if not already done"
# ./edge import -db "$DB_URL" -file "$EXAMPLE_FILE"
echo ""

echo "Step 3: Import filesystem object to database"
echo "Command: ./edge import-fs -db <DATABASE_URL> -cid $CID -store $STORE_DIR"
# ./edge import-fs -db "$DB_URL" -cid "$CID" -store "$STORE_DIR"
echo ""

echo "Step 4: Add signature to object (if signature exists in filesystem)"
echo "Command: ./edge sign -db <DATABASE_URL> -cid $CID -store $STORE_DIR"
# ./edge sign -db "$DB_URL" -cid "$CID" -store "$STORE_DIR"
echo ""

echo "Step 5: Query database using pg_graphql"
echo "Command: ./edge query -db <DATABASE_URL> -query '{objectsCollection{edges{node{cid}}}}'"
# ./edge query -db "$DB_URL" -query "{objectsCollection{edges{node{cid}}}}"
echo ""

echo "=== Workflow Complete ==="
echo ""
echo "For full end-to-end testing, start the database:"
echo "  docker-compose up -d"
echo ""
echo "Run migrations:"
echo "  psql $DB_URL -f migrations/migrations_20251029_create_tens_city_tables.sql"
echo "  psql $DB_URL -f migrations/policies_enable_rls_and_policies.sql"
echo ""
echo "Then uncomment the commands above to run the full workflow"
