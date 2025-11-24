# Tens City Examples

This directory contains example files demonstrating various features of tens-city and Petri net modeling.

## Petri Net Examples

### Simple Petri Net (`petrinet.jsonld`)

A minimal JSON-LD example demonstrating the basic structure of a Petri net document. This shows how to use Schema.org vocabulary with JSON-LD context.

**Usage:**
```bash
# Seal the document to the store
./seal -in examples/petrinet.jsonld -store data
```

### Petri Net with Inhibitor Arcs (`petrinet-inhibitor.jsonld`)

A more complex example demonstrating Petri net concepts including:
- Places with initial markings and capacities
- Transitions with multiple states
- Arcs with weights
- Inhibitor arcs (for constraint modeling)

This example uses the pflow.xyz schema vocabulary for Petri net modeling.

**Key Features:**
- `@context`: pflow.xyz schema for Petri nets
- `places`: Define states/locations in the net
- `transitions`: Define actions/events
- `arcs`: Define token flow between places and transitions
- `inhibitTransition`: Special arcs that prevent transition firing

**Usage:**
```bash
# Seal the document to the store
./seal -in examples/petrinet-inhibitor.jsonld -store data
```

## Sudoku Petri Net Example

A complete example demonstrating how to model a Sudoku puzzle using Petri nets with the `pflow-xyz/go-pflow` library.

### Location

See the `sudoku/` subdirectory for:
- `sudoku-4x4-simple.jsonld` - A 4x4 Sudoku puzzle in Petri net format
- `README.md` - Detailed documentation
- `cmd/main.go` - Go program to analyze and verify the puzzle

### Quick Start

```bash
# Navigate to the sudoku example
cd examples/sudoku

# Run the analyzer
go run cmd/main.go
```

### What You'll Learn

- How constraint satisfaction problems map to Petri nets
- Modeling game rules as places, transitions, and arcs
- Using JSON-LD for semantic Petri net representation
- Integrating go-pflow library for analysis

### Example Output

The analyzer will:
1. Display the initial puzzle state
2. Show the solution
3. Verify the solution satisfies all Sudoku constraints
4. Analyze the Petri net structure (places, transitions, arcs)

## Workflow Script

The `workflow.sh` script demonstrates a complete workflow:

1. Seal a JSON-LD file to the content-addressable store
2. Generate a CID (Content Identifier)
3. Start the web server
4. Access the sealed content

**Usage:**
```bash
./workflow.sh
```

## Further Reading

- [Petri Nets Wikipedia](https://en.wikipedia.org/wiki/Petri_net)
- [pflow.xyz](https://pflow.xyz) - Interactive Petri net editor
- [go-pflow GitHub](https://github.com/pflow-xyz/go-pflow) - Go library for Petri net simulation
- [JSON-LD](https://json-ld.org/) - JSON for Linking Data
- [Schema.org](https://schema.org/) - Structured data vocabulary

## Contributing

When adding new examples:

1. Use JSON-LD format with appropriate `@context`
2. Include clear labels and descriptions
3. Add documentation explaining the example's purpose
4. Provide usage instructions
5. Test that the example can be sealed and served

## Example Structure

Good examples should include:

- **@context**: Vocabulary definitions
- **@type**: Type of the resource
- **description**: Human-readable explanation
- **labels**: Clear naming for places/transitions
- **comments**: Inline documentation where helpful
