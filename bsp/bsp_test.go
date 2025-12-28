package bsp

import (
	"testing"

	pb "github.com/bloodmagesoftware/venture/proto/level"
)

// TestCase represents a single point collision test
type TestCase struct {
	Name        string
	Point       Point
	ExpectSolid bool // true if point should be in solid geometry
}

// TestSimpleBox tests BSP collision with a simple rectangular box
func TestSimpleBox(t *testing.T) {
	// Define a simple 10x10 box centered at origin
	// The box is solid (walls), and the interior is empty
	polygons := []Polygon{
		{
			Vertices: []Point{
				{X: -5, Y: -5},
				{X: 5, Y: -5},
				{X: 5, Y: 5},
				{X: -5, Y: 5},
			},
			IsSolid: true, // This is a solid wall
		},
	}

	// Build the BSP tree
	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	// Define test cases: points to test and their expected collision state
	testCases := []TestCase{
		{
			Name:        "Point inside box (should be empty/non-solid)",
			Point:       Point{X: 0, Y: 0},
			ExpectSolid: false,
		},
		{
			Name:        "Point outside box - far right",
			Point:       Point{X: 10, Y: 0},
			ExpectSolid: false,
		},
		{
			Name:        "Point outside box - far left",
			Point:       Point{X: -10, Y: 0},
			ExpectSolid: false,
		},
		{
			Name:        "Point outside box - far up",
			Point:       Point{X: 0, Y: 10},
			ExpectSolid: false,
		},
		{
			Name:        "Point outside box - far down",
			Point:       Point{X: 0, Y: -10},
			ExpectSolid: false,
		},
		{
			Name:        "Point on edge (right wall)",
			Point:       Point{X: 5, Y: 0},
			ExpectSolid: true,
		},
		{
			Name:        "Point near center-right",
			Point:       Point{X: 3, Y: 0},
			ExpectSolid: false,
		},
	}

	// Run all test cases
	runTestCases(t, root, testCases)
}

// TODO: Add more test cases below:

// TestLShapedRoom tests a more complex L-shaped room
func TestLShapedRoom(t *testing.T) {
	// TODO: Define an L-shaped room with multiple polygons
	// Example structure:
	//   ┌──────┐
	//   │      │
	//   │    ┌─┘
	//   │    │
	//   └────┘
	
	polygons := []Polygon{
		// TODO: Add polygon definitions for L-shaped room
	}

	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	testCases := []TestCase{
		// TODO: Add test cases for L-shaped room
	}

	runTestCases(t, root, testCases)
}

// TestMultipleRooms tests multiple separate rooms
func TestMultipleRooms(t *testing.T) {
	// TODO: Define multiple separate rectangular rooms
	polygons := []Polygon{
		// TODO: Add polygon definitions
	}

	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	testCases := []TestCase{
		// TODO: Add test cases
	}

	runTestCases(t, root, testCases)
}

// TestNestedBoxes tests boxes within boxes
func TestNestedBoxes(t *testing.T) {
	// TODO: Define nested boxes (e.g., a room with a pillar in the middle)
	polygons := []Polygon{
		// TODO: Add polygon definitions
	}

	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	testCases := []TestCase{
		// TODO: Add test cases
	}

	runTestCases(t, root, testCases)
}

// TestConvexPolygon tests a convex polygon (e.g., hexagon)
func TestConvexPolygon(t *testing.T) {
	// TODO: Define a convex polygon like a hexagon
	polygons := []Polygon{
		// TODO: Add polygon definitions
	}

	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	testCases := []TestCase{
		// TODO: Add test cases
	}

	runTestCases(t, root, testCases)
}

// TestConcavePolygon tests a concave polygon (will need to be split)
func TestConcavePolygon(t *testing.T) {
	// TODO: Define a concave polygon (e.g., star shape or U-shape)
	polygons := []Polygon{
		// TODO: Add polygon definitions
	}

	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	testCases := []TestCase{
		// TODO: Add test cases
	}

	runTestCases(t, root, testCases)
}

// TestEmptySpace tests behavior with no geometry
func TestEmptySpace(t *testing.T) {
	// TODO: Test with no polygons or all non-solid polygons
	polygons := []Polygon{}

	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	testCases := []TestCase{
		// TODO: Add test cases
	}

	runTestCases(t, root, testCases)
}

// TestCorridors tests narrow corridors between rooms
func TestCorridors(t *testing.T) {
	// TODO: Define rooms connected by narrow corridors
	polygons := []Polygon{
		// TODO: Add polygon definitions
	}

	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	testCases := []TestCase{
		// TODO: Add test cases
	}

	runTestCases(t, root, testCases)
}

// Helper Functions

// runTestCases runs all test cases against the BSP tree
func runTestCases(t *testing.T, root *pb.BSPNode, cases []TestCase) {
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			result := PointInBSP(root, tc.Point)
			if result != tc.ExpectSolid {
				t.Errorf("Point (%f, %f): expected solid=%v, got solid=%v",
					tc.Point.X, tc.Point.Y, tc.ExpectSolid, result)
			}
		})
	}
}

