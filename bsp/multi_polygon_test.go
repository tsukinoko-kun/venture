package bsp

import (
	"testing"
)

// TestMultiPolygonBug is a focused test for debugging multi-polygon issues
// This tests specific scenarios that might be failing in the level editor
func TestMultiPolygonBug(t *testing.T) {
	// Two boxes that don't overlap at all - testing if second box is ignored
	t.Run("Two separate boxes far apart", func(t *testing.T) {
		polygons := []Polygon{
			// Box 1: (-10, -10) to (-5, -5)
			{
				Vertices: []Point{
					{X: -10, Y: -10},
					{X: -5, Y: -10},
					{X: -5, Y: -5},
					{X: -10, Y: -5},
				},
				IsSolid: true,
			},
			// Box 2: (5, 5) to (10, 10)
			{
				Vertices: []Point{
					{X: 5, Y: 5},
					{X: 10, Y: 5},
					{X: 10, Y: 10},
					{X: 5, Y: 10},
				},
				IsSolid: true,
			},
		}

		builder := NewBSPBuilder(polygons)
		levelData := builder.Build()

		// Test point in first box
		p1 := Point{X: -7, Y: -7}
		if !PointInBSP(levelData.Nodes, levelData.RootIndex, p1) {
			t.Errorf("Point (%v, %v) should be SOLID (in first box)", p1.X, p1.Y)
		}

		// Test point in second box - THIS IS THE KEY TEST
		p2 := Point{X: 7, Y: 7}
		if !PointInBSP(levelData.Nodes, levelData.RootIndex, p2) {
			t.Errorf("Point (%v, %v) should be SOLID (in second box) - SECOND BOX BEING IGNORED!", p2.X, p2.Y)
		}

		// Test point between boxes (should be empty)
		p3 := Point{X: 0, Y: 0}
		if PointInBSP(levelData.Nodes, levelData.RootIndex, p3) {
			t.Errorf("Point (%v, %v) should be EMPTY (between boxes)", p3.X, p3.Y)
		}

		// Test point outside both
		p4 := Point{X: 20, Y: 20}
		if PointInBSP(levelData.Nodes, levelData.RootIndex, p4) {
			t.Errorf("Point (%v, %v) should be EMPTY (outside both boxes)", p4.X, p4.Y)
		}
	})

	// Three boxes in different quadrants
	t.Run("Three boxes in different quadrants", func(t *testing.T) {
		polygons := []Polygon{
			// Box 1: bottom-left quadrant
			{
				Vertices: []Point{
					{X: -10, Y: -10},
					{X: -5, Y: -10},
					{X: -5, Y: -5},
					{X: -10, Y: -5},
				},
				IsSolid: true,
			},
			// Box 2: bottom-right quadrant
			{
				Vertices: []Point{
					{X: 5, Y: -10},
					{X: 10, Y: -10},
					{X: 10, Y: -5},
					{X: 5, Y: -5},
				},
				IsSolid: true,
			},
			// Box 3: top-right quadrant
			{
				Vertices: []Point{
					{X: 5, Y: 5},
					{X: 10, Y: 5},
					{X: 10, Y: 10},
					{X: 5, Y: 10},
				},
				IsSolid: true,
			},
		}

		builder := NewBSPBuilder(polygons)
		levelData := builder.Build()

		tests := []struct {
			name        string
			point       Point
			expectSolid bool
		}{
			{"In box 1", Point{X: -7, Y: -7}, true},
			{"In box 2", Point{X: 7, Y: -7}, true},
			{"In box 3", Point{X: 7, Y: 7}, true},
			{"Origin (empty)", Point{X: 0, Y: 0}, false},
			{"Top-left (empty)", Point{X: -7, Y: 7}, false},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result := PointInBSP(levelData.Nodes, levelData.RootIndex, tc.point)
				if result != tc.expectSolid {
					t.Errorf("Point (%v, %v): got solid=%v, want solid=%v",
						tc.point.X, tc.point.Y, result, tc.expectSolid)
				}
			})
		}
	})

	// Test with boxes arranged horizontally (same Y range)
	t.Run("Horizontal row of boxes", func(t *testing.T) {
		polygons := []Polygon{
			{
				Vertices: []Point{
					{X: 0, Y: 0},
					{X: 2, Y: 0},
					{X: 2, Y: 2},
					{X: 0, Y: 2},
				},
				IsSolid: true,
			},
			{
				Vertices: []Point{
					{X: 5, Y: 0},
					{X: 7, Y: 0},
					{X: 7, Y: 2},
					{X: 5, Y: 2},
				},
				IsSolid: true,
			},
			{
				Vertices: []Point{
					{X: 10, Y: 0},
					{X: 12, Y: 0},
					{X: 12, Y: 2},
					{X: 10, Y: 2},
				},
				IsSolid: true,
			},
		}

		builder := NewBSPBuilder(polygons)
		levelData := builder.Build()

		tests := []struct {
			name        string
			point       Point
			expectSolid bool
		}{
			{"In box 1", Point{X: 1, Y: 1}, true},
			{"In box 2", Point{X: 6, Y: 1}, true},
			{"In box 3", Point{X: 11, Y: 1}, true},
			{"Gap between 1 and 2", Point{X: 3, Y: 1}, false},
			{"Gap between 2 and 3", Point{X: 8, Y: 1}, false},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result := PointInBSP(levelData.Nodes, levelData.RootIndex, tc.point)
				if result != tc.expectSolid {
					t.Errorf("Point (%v, %v): got solid=%v, want solid=%v",
						tc.point.X, tc.point.Y, result, tc.expectSolid)
				}
			})
		}
	})
}

// TestIndividualPolygonsVsMerged tests each polygon independently vs merged
func TestIndividualPolygonsVsMerged(t *testing.T) {
	// Define two separate boxes
	poly1 := Polygon{
		Vertices: []Point{
			{X: 0, Y: 0},
			{X: 2, Y: 0},
			{X: 2, Y: 2},
			{X: 0, Y: 2},
		},
		IsSolid: true,
	}

	poly2 := Polygon{
		Vertices: []Point{
			{X: 10, Y: 10},
			{X: 12, Y: 10},
			{X: 12, Y: 12},
			{X: 10, Y: 12},
		},
		IsSolid: true,
	}

	// Build individual trees
	builder1 := NewBSPBuilder([]Polygon{poly1})
	tree1 := builder1.Build()

	builder2 := NewBSPBuilder([]Polygon{poly2})
	tree2 := builder2.Build()

	// Build merged tree
	builderBoth := NewBSPBuilder([]Polygon{poly1, poly2})
	treeBoth := builderBoth.Build()

	// Test point in poly1
	p1 := Point{X: 1, Y: 1}
	t.Run("Point in poly1 - individual tree", func(t *testing.T) {
		if !PointInBSP(tree1.Nodes, tree1.RootIndex, p1) {
			t.Error("Should be solid in tree1")
		}
	})
	t.Run("Point in poly1 - merged tree", func(t *testing.T) {
		if !PointInBSP(treeBoth.Nodes, treeBoth.RootIndex, p1) {
			t.Error("Should be solid in merged tree")
		}
	})

	// Test point in poly2
	p2 := Point{X: 11, Y: 11}
	t.Run("Point in poly2 - individual tree", func(t *testing.T) {
		if !PointInBSP(tree2.Nodes, tree2.RootIndex, p2) {
			t.Error("Should be solid in tree2")
		}
	})
	t.Run("Point in poly2 - merged tree", func(t *testing.T) {
		if !PointInBSP(treeBoth.Nodes, treeBoth.RootIndex, p2) {
			t.Error("Should be solid in merged tree - BUG: SECOND POLYGON IGNORED!")
		}
	})

	// Test point between polys
	p3 := Point{X: 5, Y: 5}
	t.Run("Point between - individual tree1", func(t *testing.T) {
		if PointInBSP(tree1.Nodes, tree1.RootIndex, p3) {
			t.Error("Should be empty in tree1")
		}
	})
	t.Run("Point between - individual tree2", func(t *testing.T) {
		if PointInBSP(tree2.Nodes, tree2.RootIndex, p3) {
			t.Error("Should be empty in tree2")
		}
	})
	t.Run("Point between - merged tree", func(t *testing.T) {
		if PointInBSP(treeBoth.Nodes, treeBoth.RootIndex, p3) {
			t.Error("Should be empty in merged tree")
		}
	})
}
