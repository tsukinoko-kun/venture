# BSP-Tree TDD Quick Start

## What You Have

A complete TDD setup for implementing BSP-Tree collision detection:

1. ✓ Protobuf definition (`proto/level.proto`)
2. ✓ Compilation script (`scripts/compile_proto.sh`)
3. ✓ Generated Go code (`proto/level/level.pb.go`)
4. ✓ Core package with helpers (`bsp/bsp.go`)
5. ✓ Test suite with sample test (`bsp/bsp_test.go`)
6. ✓ Documentation (`bsp/README.md`, `bsp/SETUP_SUMMARY.md`)

## Quick Commands

```bash
# Run all tests
go test ./bsp -v

# Run specific test
go test ./bsp -v -run TestSimpleBox

# Build the package
go build ./bsp

# Recompile protobuf (after changes)
./scripts/compile_proto.sh
```

## What to Implement

**Main task**: Implement `BSPBuilder.Build()` in `bsp/bsp.go`

Current stub (line ~76):
```go
func (b *BSPBuilder) Build() *pb.BSPNode {
    // TODO: Implement BSP tree construction
    // For now, return a simple leaf node
    return &pb.BSPNode{
        Type: &pb.BSPNode_Leaf{
            Leaf: &pb.Leaf{
                SectorId:       0,
                PolygonIndices: []int32{},
                IsSolid:        false,
            },
        },
    }
}
```

## Implementation Strategy

### Step 1: Make the simplest test pass
Start with `TestSimpleBox` - it has 7 test cases for a simple rectangular box.

### Step 2: Basic Algorithm
```
1. Base case: If all polygons are similar (all solid or all empty), return leaf
2. Select partition: Pick a line from one of the polygon edges
3. Split polygons: Classify polygons as front/back/spanning
4. Recurse: Build front and back subtrees
5. Return split node with partition line and children
```

### Step 3: Add more test cases
Fill in the template tests in `bsp_test.go`:
- `TestLShapedRoom`
- `TestMultipleRooms`
- `TestNestedBoxes`
- etc.

## Helper Functions Available

**Geometry:**
- `Vector2.Normalize()` - Normalize vector
- `Vector2.Dot(other)` - Dot product
- `Line.PointSide(p)` - Signed distance from point to line
- `Line.ClassifyPoint(p)` - Returns 1 (front), -1 (back), 0 (on line)

**Node Creation:**
- `NewLeafNode(sectorID, indices, isSolid)` - Create leaf
- `NewSplitNode(nx, ny, dist, front, back)` - Create split node

**Testing:**
- `PointInBSP(node, point)` - Query the tree (already implemented!)

## Example Test Case Structure

```go
func TestMyGeometry(t *testing.T) {
    // 1. Define geometry
    polygons := []Polygon{
        {
            Vertices: []Point{
                {X: 0, Y: 0},
                {X: 10, Y: 0},
                {X: 10, Y: 10},
                {X: 0, Y: 10},
            },
            IsSolid: true,
        },
    }

    // 2. Build BSP tree
    builder := NewBSPBuilder(polygons)
    root := builder.Build()

    // 3. Define test points
    testCases := []TestCase{
        {
            Name:        "Point inside",
            Point:       Point{X: 5, Y: 5},
            ExpectSolid: false, // Interior is empty
        },
        {
            Name:        "Point outside",
            Point:       Point{X: 15, Y: 15},
            ExpectSolid: false,
        },
    }

    // 4. Run tests
    runTestCases(t, root, testCases)
}
```

## Current Test Status

Running `go test ./bsp -v -run TestSimpleBox`:

```
=== RUN   TestSimpleBox/Point_inside_box_(should_be_empty/non-solid)
--- PASS

=== RUN   TestSimpleBox/Point_on_edge_(right_wall)
    bsp_test.go:213: Point (5.000000, 0.000000): expected solid=true, got solid=false
--- FAIL
```

Most tests pass with the stub (because they expect `false`), but edge detection fails - **this is expected!**

## TDD Workflow

1. **RED**: Tests fail (✓ current state)
2. **GREEN**: Implement `Build()` to make tests pass
3. **REFACTOR**: Optimize and clean up implementation
4. **REPEAT**: Add more test cases, improve algorithm

## Tips

- Start simple: Even a naive implementation that handles one polygon is progress
- Use `NewLeafNode()` and `NewSplitNode()` helpers
- The `PointInBSP()` function is fully working - it will correctly test your tree
- Print debug info during implementation: `fmt.Printf("Splitting at: %v\n", line)`
- Add visualization if needed (draw the BSP tree structure)

## Get Started

```bash
# Open the implementation file
open bsp/bsp.go

# Open the test file
open bsp/bsp_test.go

# Run tests in watch mode (if you have a tool)
go test ./bsp -v -run TestSimpleBox
```

Good luck! 🚀

