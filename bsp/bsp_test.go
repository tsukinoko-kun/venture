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
	// The box is a solid obstacle - points inside should collide
	polygons := []Polygon{
		{
			Vertices: []Point{
				{X: -5, Y: -5},
				{X: 5, Y: -5},
				{X: 5, Y: 5},
				{X: -5, Y: 5},
			},
			IsSolid: true, // This is a solid obstacle
		},
	}

	// Build the BSP tree
	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	// Define test cases: points to test and their expected collision state
	testCases := []TestCase{
		{
			Name:        "Point inside box (should be solid)",
			Point:       Point{X: 0, Y: 0},
			ExpectSolid: true,
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
			Name:        "Point near center-right (inside)",
			Point:       Point{X: 3, Y: 0},
			ExpectSolid: true,
		},
	}

	// Run all test cases
	runTestCases(t, root, testCases)
}

// TODO: Add more test cases below:

// TestLShapedRoom tests a more complex L-shaped room
func TestLShapedRoom(t *testing.T) {
	// L-shaped concave polygon
	//   0,4 ---- 2,4
	//     |       |
	//     |   2,2 +
	//     |       |
	//   0,0 ---- 4,0
	
	polygons := []Polygon{
		{
			Vertices: []Point{
				{X: 0, Y: 0},
				{X: 4, Y: 0},
				{X: 4, Y: 2},
				{X: 2, Y: 2},
				{X: 2, Y: 4},
				{X: 0, Y: 4},
			},
			IsSolid: true,
		},
	}

	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	testCases := []TestCase{
		{
			Name:        "Point in lower part of L",
			Point:       Point{X: 1, Y: 1},
			ExpectSolid: true,
		},
		{
			Name:        "Point in upper part of L",
			Point:       Point{X: 1, Y: 3},
			ExpectSolid: true,
		},
		{
			Name:        "Point in concave corner (outside)",
			Point:       Point{X: 3, Y: 3},
			ExpectSolid: false,
		},
		{
			Name:        "Point far outside",
			Point:       Point{X: 10, Y: 10},
			ExpectSolid: false,
		},
		{
			Name:        "Point on outer edge",
			Point:       Point{X: 0, Y: 2},
			ExpectSolid: true,
		},
	}

	runTestCases(t, root, testCases)
}

// TestMultipleRooms tests multiple separate rooms
func TestMultipleRooms(t *testing.T) {
	// Two separate rectangular obstacles
	polygons := []Polygon{
		// First box at (0,0) to (2,2)
		{
			Vertices: []Point{
				{X: 0, Y: 0},
				{X: 2, Y: 0},
				{X: 2, Y: 2},
				{X: 0, Y: 2},
			},
			IsSolid: true,
		},
		// Second box at (5,5) to (7,7)
		{
			Vertices: []Point{
				{X: 5, Y: 5},
				{X: 7, Y: 5},
				{X: 7, Y: 7},
				{X: 5, Y: 7},
			},
			IsSolid: true,
		},
	}

	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	testCases := []TestCase{
		{
			Name:        "Point in first box",
			Point:       Point{X: 1, Y: 1},
			ExpectSolid: true,
		},
		{
			Name:        "Point in second box",
			Point:       Point{X: 6, Y: 6},
			ExpectSolid: true,
		},
		{
			Name:        "Point between boxes",
			Point:       Point{X: 3, Y: 3},
			ExpectSolid: false,
		},
		{
			Name:        "Point outside both boxes",
			Point:       Point{X: 10, Y: 10},
			ExpectSolid: false,
		},
	}

	runTestCases(t, root, testCases)
}

// TestNestedBoxes tests boxes within boxes
func TestNestedBoxes(t *testing.T) {
	// Large box with a smaller box inside (like a room with a pillar)
	polygons := []Polygon{
		// Outer box (10x10)
		{
			Vertices: []Point{
				{X: 0, Y: 0},
				{X: 10, Y: 0},
				{X: 10, Y: 10},
				{X: 0, Y: 10},
			},
			IsSolid: true,
		},
		// Inner box (2x2) - pillar in the center
		{
			Vertices: []Point{
				{X: 4, Y: 4},
				{X: 6, Y: 4},
				{X: 6, Y: 6},
				{X: 4, Y: 6},
			},
			IsSolid: true,
		},
	}

	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	testCases := []TestCase{
		{
			Name:        "Point in outer box (not in inner)",
			Point:       Point{X: 1, Y: 1},
			ExpectSolid: true,
		},
		{
			Name:        "Point in inner box",
			Point:       Point{X: 5, Y: 5},
			ExpectSolid: true,
		},
		{
			Name:        "Point outside both boxes",
			Point:       Point{X: 15, Y: 15},
			ExpectSolid: false,
		},
		{
			Name:        "Point between boxes",
			Point:       Point{X: 2, Y: 5},
			ExpectSolid: true,
		},
	}

	runTestCases(t, root, testCases)
}

// TestConvexPolygon tests a convex polygon (e.g., hexagon)
func TestConvexPolygon(t *testing.T) {
	// Regular hexagon centered at origin with radius ~5
	polygons := []Polygon{
		{
			Vertices: []Point{
				{X: 5, Y: 0},
				{X: 2.5, Y: 4.33},
				{X: -2.5, Y: 4.33},
				{X: -5, Y: 0},
				{X: -2.5, Y: -4.33},
				{X: 2.5, Y: -4.33},
			},
			IsSolid: true,
		},
	}

	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	testCases := []TestCase{
		{
			Name:        "Point at center",
			Point:       Point{X: 0, Y: 0},
			ExpectSolid: true,
		},
		{
			Name:        "Point inside near edge",
			Point:       Point{X: 3, Y: 0},
			ExpectSolid: true,
		},
		{
			Name:        "Point outside",
			Point:       Point{X: 10, Y: 10},
			ExpectSolid: false,
		},
		{
			Name:        "Point on vertex",
			Point:       Point{X: 5, Y: 0},
			ExpectSolid: true,
		},
	}

	runTestCases(t, root, testCases)
}

// TestConcavePolygon tests a concave polygon (will need to be split)
func TestConcavePolygon(t *testing.T) {
	// Star-like concave polygon (CGAL will partition this into convex pieces)
	// Simple U-shape
	polygons := []Polygon{
		{
			Vertices: []Point{
				{X: 0, Y: 0},
				{X: 0, Y: 4},
				{X: 2, Y: 4},
				{X: 2, Y: 1},
				{X: 4, Y: 1},
				{X: 4, Y: 4},
				{X: 6, Y: 4},
				{X: 6, Y: 0},
			},
			IsSolid: true,
		},
	}

	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	testCases := []TestCase{
		{
			Name:        "Point in left arm of U",
			Point:       Point{X: 1, Y: 2},
			ExpectSolid: true,
		},
		{
			Name:        "Point in right arm of U",
			Point:       Point{X: 5, Y: 2},
			ExpectSolid: true,
		},
		{
			Name:        "Point in the U gap (concave part)",
			Point:       Point{X: 3, Y: 2},
			ExpectSolid: false,
		},
		{
			Name:        "Point in bottom connecting part",
			Point:       Point{X: 3, Y: 0.5},
			ExpectSolid: true,
		},
		{
			Name:        "Point outside",
			Point:       Point{X: 10, Y: 10},
			ExpectSolid: false,
		},
	}

	runTestCases(t, root, testCases)
}

// TestEmptySpace tests behavior with no geometry
func TestEmptySpace(t *testing.T) {
	// No polygons - entire space should be non-solid
	polygons := []Polygon{}

	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	testCases := []TestCase{
		{
			Name:        "Point at origin",
			Point:       Point{X: 0, Y: 0},
			ExpectSolid: false,
		},
		{
			Name:        "Point anywhere",
			Point:       Point{X: 100, Y: 100},
			ExpectSolid: false,
		},
		{
			Name:        "Negative coordinates",
			Point:       Point{X: -50, Y: -50},
			ExpectSolid: false,
		},
	}

	runTestCases(t, root, testCases)
}

// TestCorridors tests narrow corridors between rooms
func TestCorridors(t *testing.T) {
	// Three boxes forming a corridor pattern
	// [Box1] - (corridor) - [Box2]
	polygons := []Polygon{
		// Left box
		{
			Vertices: []Point{
				{X: 0, Y: 0},
				{X: 2, Y: 0},
				{X: 2, Y: 2},
				{X: 0, Y: 2},
			},
			IsSolid: true,
		},
		// Horizontal corridor
		{
			Vertices: []Point{
				{X: 2, Y: 0.5},
				{X: 4, Y: 0.5},
				{X: 4, Y: 1.5},
				{X: 2, Y: 1.5},
			},
			IsSolid: true,
		},
		// Right box
		{
			Vertices: []Point{
				{X: 4, Y: 0},
				{X: 6, Y: 0},
				{X: 6, Y: 2},
				{X: 4, Y: 2},
			},
			IsSolid: true,
		},
	}

	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	testCases := []TestCase{
		{
			Name:        "Point in left box",
			Point:       Point{X: 1, Y: 1},
			ExpectSolid: true,
		},
		{
			Name:        "Point in corridor",
			Point:       Point{X: 3, Y: 1},
			ExpectSolid: true,
		},
		{
			Name:        "Point in right box",
			Point:       Point{X: 5, Y: 1},
			ExpectSolid: true,
		},
		{
			Name:        "Point outside above corridor",
			Point:       Point{X: 3, Y: 3},
			ExpectSolid: false,
		},
		{
			Name:        "Point below corridor entrance",
			Point:       Point{X: 3, Y: 0.2},
			ExpectSolid: false,
		},
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

