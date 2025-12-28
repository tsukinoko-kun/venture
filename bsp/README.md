# BSP Tree Package

This package provides BSP (Binary Space Partitioning) tree functionality for 2D collision detection in the Venture game engine.

## Overview

The BSP tree is used to efficiently determine point-in-polygon tests for collision detection. The tree structure is stored in Protocol Buffers format for compact storage and fast loading.

## Files

- `bsp.go` - Core BSP tree data structures and helper functions
- `bsp_test.go` - TDD test suite for BSP tree construction
- `../proto/level.proto` - Protocol Buffer definition for BSP tree storage
- `../proto/level/level.pb.go` - Generated Go code from protobuf (auto-generated)
- `../scripts/compile_proto.sh` - Script to compile protobuf definitions

## Data Structures

### Core Types

- `Point` - Represents a 2D point (X, Y)
- `Polygon` - Represents a collision polygon with vertices and solid state
- `Vector2` - 2D vector with utility methods (Normalize, Dot)
- `Line` - 2D line using plane equation: Normal · Point = Distance
- `BSPBuilder` - Builder for constructing BSP trees from polygons

### Protocol Buffer Messages

- `LevelData` - Top-level message containing the BSP tree root
- `BSPNode` - A node in the tree (either Split or Leaf)
- `Split` - Interior node with a splitting plane and two children
- `Leaf` - Leaf node containing sector information and solid state

## Helper Functions

### Geometry Helpers

- `Vector2.Normalize()` - Returns normalized vector
- `Vector2.Dot(other)` - Computes dot product
- `Line.PointSide(p)` - Returns signed distance from point to line
- `Line.ClassifyPoint(p)` - Returns 1 (front), -1 (back), or 0 (on line)

### BSP Construction

- `NewBSPBuilder(polygons)` - Creates a new BSP builder
- `BSPBuilder.Build()` - Constructs and returns the BSP tree root (TODO: implement)

### BSP Query

- `PointInBSP(node, point)` - Tests if a point is inside solid geometry

### Node Creation

- `NewLeafNode(sectorID, polygonIndices, isSolid)` - Creates a leaf node
- `NewSplitNode(normalX, normalY, distance, front, back)` - Creates a split node
- `NewLevelData(root)` - Creates a level data protobuf message

## Testing

The test file (`bsp_test.go`) provides a TDD framework with multiple test scenarios:

### Running Tests

```bash
# Run all BSP tests
go test ./bsp -v

# Run a specific test
go test ./bsp -v -run TestSimpleBox
```

### Test Structure

Each test follows this pattern:

1. Define polygons representing the level geometry
2. Create a BSP builder and build the tree
3. Define test cases with points and expected collision results
4. Run test cases through `runTestCases()` helper

### Sample Test

```go
func TestSimpleBox(t *testing.T) {
    // Define geometry
    polygons := []Polygon{
        {
            Vertices: []Point{
                {X: -5, Y: -5},
                {X: 5, Y: -5},
                {X: 5, Y: 5},
                {X: -5, Y: 5},
            },
            IsSolid: true,
        },
    }

    // Build BSP tree
    builder := NewBSPBuilder(polygons)
    root := builder.Build()

    // Define test points
    testCases := []TestCase{
        {
            Name:        "Point inside box",
            Point:       Point{X: 0, Y: 0},
            ExpectSolid: false, // Interior is empty
        },
        // ... more test cases
    }

    runTestCases(t, root, testCases)
}
```

### Prepared Test Templates

The following test templates are ready for implementation:

- `TestSimpleBox` - ✓ Sample test with a simple rectangular box
- `TestLShapedRoom` - TODO: L-shaped room geometry
- `TestMultipleRooms` - TODO: Multiple separate rooms
- `TestNestedBoxes` - TODO: Boxes within boxes (e.g., pillar in room)
- `TestConvexPolygon` - TODO: Convex polygons (hexagon, etc.)
- `TestConcavePolygon` - TODO: Concave polygons that need splitting
- `TestEmptySpace` - TODO: No geometry or all non-solid
- `TestCorridors` - TODO: Narrow corridors between rooms

## Protocol Buffer Compilation

To recompile the protocol buffer definitions after making changes:

```bash
# Requires protoc and protoc-gen-go to be installed
# Install protoc-gen-go:
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# Compile the proto files:
./scripts/compile_proto.sh
```

## Implementation Notes

### Current State

The package is set up for Test-Driven Development (TDD):

- ✓ Protocol buffer definitions created
- ✓ Compilation script working
- ✓ Helper functions and data structures implemented
- ✓ Test infrastructure ready
- ✓ Sample test provided
- ⚠ BSP tree construction (`BSPBuilder.Build()`) returns stub - **NEEDS IMPLEMENTATION**

### Next Steps

1. Implement `BSPBuilder.Build()` to construct the actual BSP tree
2. Add polygon splitting logic for handling polygons that cross partition lines
3. Implement partition selection heuristics (minimize splits, balance tree)
4. Fill in the TODO test cases with actual geometry and test points
5. Test edge cases (degenerate polygons, collinear edges, etc.)

### BSP Algorithm Overview (for implementation)

The BSP construction algorithm should:

1. **Base Case**: If all polygons are in the same region (all solid or all empty), create a leaf node
2. **Select Partition**: Choose a splitting line from one of the polygon edges
3. **Split Polygons**: Divide polygons based on which side of the partition they're on
4. **Recurse**: Build front and back subtrees with their respective polygon sets
5. **Create Node**: Return a split node with the partition line and child nodes

### Partition Selection Heuristics

Good partition selection minimizes:
- Number of polygon splits
- Tree depth (keep balanced)
- Overlapping geometry

Common strategies:
- Use edges from existing polygons as partition candidates
- Score candidates based on split count and balance
- Use alternating X/Y axis splits for simple balanced trees

## Usage Example

```go
// Define level geometry
polygons := []Polygon{
    {
        Vertices: []Point{
            {X: 0, Y: 0},
            {X: 100, Y: 0},
            {X: 100, Y: 100},
            {X: 0, Y: 100},
        },
        IsSolid: true,
    },
}

// Build BSP tree
builder := bsp.NewBSPBuilder(polygons)
root := builder.Build()

// Create level data for serialization
levelData := bsp.NewLevelData(root)

// Query collision
point := bsp.Point{X: 50, Y: 50}
isInSolid := bsp.PointInBSP(root, point)
```

## Building the CGAL Wrapper

This package uses CGAL for convex polygon decomposition. The C++ code is compiled into a static library that is linked into the Go binary.

### Requirements

- CGAL:
  - macOS: `brew install cgal`
  - Linux: `apt-get install libcgal-dev`
  - Windows: Install via MSYS2: `pacman -S mingw-w64-x86_64-cgal mingw-w64-x86_64-gmp`
- GMP: Usually installed with CGAL

### Building

```bash
cd cgal
make
```

This creates `libpartition.a` which is statically linked into the Go binary.

### Running Tests

```bash
go test ./bsp -v
```

Or use the helper script:

```bash
./run_tests.sh -v
```
