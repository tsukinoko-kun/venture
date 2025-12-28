package bsp

import (
	"testing"
)

// TestLevelPolygons tests with the exact polygons from the test.yaml level file
func TestLevelPolygons(t *testing.T) {
	// These are the exact polygons from levels/test.yaml
	
	// Polygon 1: Complex L-shaped (concave)
	poly1 := Polygon{
		Vertices: []Point{
			{X: -2, Y: -1},
			{X: -2, Y: -7},
			{X: -1, Y: -7},
			{X: -1, Y: -6},
			{X: 0, Y: -6},
			{X: 0, Y: -5},
			{X: 1, Y: -5},
			{X: 1, Y: -4},
			{X: 2, Y: -4},
			{X: 2, Y: -1},
			{X: 1, Y: -1},
			{X: 1, Y: -3},
			{X: -1, Y: -3},
			{X: -1, Y: -1},
		},
		IsSolid: true,
	}
	
	// Polygon 2: Rectangle at bottom (-3,-1) to (3,0)
	poly2 := Polygon{
		Vertices: []Point{
			{X: -3, Y: -1},
			{X: -3, Y: 0},
			{X: 3, Y: 0},
			{X: 3, Y: -1},
		},
		IsSolid: true,
	}
	
	// Polygon 3: Rectangle at top (-3,0) to (3,2)
	poly3 := Polygon{
		Vertices: []Point{
			{X: -3, Y: 2},
			{X: 3, Y: 2},
			{X: 3, Y: 0},
			{X: -3, Y: 0},
		},
		IsSolid: true,
	}

	t.Run("All three polygons", func(t *testing.T) {
		polygons := []Polygon{poly1, poly2, poly3}
		builder := NewBSPBuilder(polygons)
		root := builder.Build()
		
		tests := []struct {
			name        string
			point       Point
			expectSolid bool
		}{
			// Points in polygon 1 (L-shaped)
			{"In poly1 - top left", Point{X: -1.5, Y: -2}, true},
			{"In poly1 - bottom", Point{X: -1.5, Y: -6.5}, true},
			
			// Points in polygon 2 (bottom rectangle)
			{"In poly2 - center", Point{X: 0, Y: -0.5}, true},
			{"In poly2 - left side", Point{X: -2.5, Y: -0.5}, true},
			{"In poly2 - right side", Point{X: 2.5, Y: -0.5}, true},
			
			// Points in polygon 3 (top rectangle)
			{"In poly3 - center", Point{X: 0, Y: 1}, true},
			{"In poly3 - left side", Point{X: -2.5, Y: 1}, true},
			{"In poly3 - right side", Point{X: 2.5, Y: 1}, true},
			
			// Points outside all polygons
			{"Outside - far left", Point{X: -5, Y: 0}, false},
			{"Outside - far right", Point{X: 5, Y: 0}, false},
			{"Outside - far up", Point{X: 0, Y: 5}, false},
			{"Outside - far down", Point{X: 0, Y: -10}, false},
		}
		
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result := PointInBSP(root, tc.point)
				if result != tc.expectSolid {
					t.Errorf("Point (%v, %v): got solid=%v, want solid=%v",
						tc.point.X, tc.point.Y, result, tc.expectSolid)
				}
			})
		}
	})

	// Test each polygon individually to verify they work on their own
	t.Run("Polygon 1 only", func(t *testing.T) {
		builder := NewBSPBuilder([]Polygon{poly1})
		root := builder.Build()
		
		p := Point{X: -1.5, Y: -2}
		if !PointInBSP(root, p) {
			t.Errorf("Point in poly1 should be solid")
		}
		
		pOut := Point{X: 5, Y: 0}
		if PointInBSP(root, pOut) {
			t.Errorf("Point outside poly1 should be empty")
		}
	})
	
	t.Run("Polygon 2 only", func(t *testing.T) {
		builder := NewBSPBuilder([]Polygon{poly2})
		root := builder.Build()
		
		p := Point{X: 0, Y: -0.5}
		if !PointInBSP(root, p) {
			t.Errorf("Point in poly2 should be solid")
		}
		
		pOut := Point{X: 0, Y: 5}
		if PointInBSP(root, pOut) {
			t.Errorf("Point outside poly2 should be empty")
		}
	})
	
	t.Run("Polygon 3 only", func(t *testing.T) {
		builder := NewBSPBuilder([]Polygon{poly3})
		root := builder.Build()
		
		p := Point{X: 0, Y: 1}
		if !PointInBSP(root, p) {
			t.Errorf("Point in poly3 should be solid")
		}
		
		pOut := Point{X: 0, Y: -5}
		if PointInBSP(root, pOut) {
			t.Errorf("Point outside poly3 should be empty")
		}
	})
	
	// Test progressive addition
	t.Run("Poly 1 + Poly 2", func(t *testing.T) {
		builder := NewBSPBuilder([]Polygon{poly1, poly2})
		root := builder.Build()
		
		// Point in poly1
		p1 := Point{X: -1.5, Y: -2}
		if !PointInBSP(root, p1) {
			t.Errorf("Point in poly1 should be solid when merged with poly2")
		}
		
		// Point in poly2
		p2 := Point{X: 0, Y: -0.5}
		if !PointInBSP(root, p2) {
			t.Errorf("Point in poly2 should be solid when merged with poly1")
		}
	})
	
	t.Run("Poly 2 + Poly 3", func(t *testing.T) {
		builder := NewBSPBuilder([]Polygon{poly2, poly3})
		root := builder.Build()
		
		// Point in poly2
		p2 := Point{X: 0, Y: -0.5}
		if !PointInBSP(root, p2) {
			t.Errorf("Point in poly2 should be solid when merged with poly3")
		}
		
		// Point in poly3
		p3 := Point{X: 0, Y: 1}
		if !PointInBSP(root, p3) {
			t.Errorf("Point in poly3 should be solid when merged with poly2")
		}
	})
}

