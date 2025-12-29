package bsp

import (
	"testing"
)

func TestCGALPartition(t *testing.T) {
	t.Run("Convex square", func(t *testing.T) {
		// A simple square should return as-is
		square := Polygon{
			Vertices: []Point{
				{X: 0, Y: 0},
				{X: 10, Y: 0},
				{X: 10, Y: 10},
				{X: 0, Y: 10},
			},
			IsSolid: true,
		}

		result, err := PartitionPolygonConvex(square)
		if err != nil {
			t.Fatalf("Failed to partition square: %v", err)
		}

		if len(result) != 1 {
			t.Errorf("Expected 1 polygon, got %d", len(result))
		}

		if len(result[0].Vertices) != 4 {
			t.Errorf("Expected 4 vertices, got %d", len(result[0].Vertices))
		}
	})

	t.Run("L-shaped concave polygon", func(t *testing.T) {
		// An L-shape should be partitioned into multiple convex polygons
		lShape := Polygon{
			Vertices: []Point{
				{X: 0, Y: 0},
				{X: 4, Y: 0},
				{X: 4, Y: 2},
				{X: 2, Y: 2},
				{X: 2, Y: 4},
				{X: 0, Y: 4},
			},
			IsSolid: true,
		}

		result, err := PartitionPolygonConvex(lShape)
		if err != nil {
			t.Fatalf("Failed to partition L-shape: %v", err)
		}

		if len(result) < 2 {
			t.Errorf("Expected at least 2 polygons for L-shape, got %d", len(result))
		}

		// Verify all results maintain the solid flag
		for i, poly := range result {
			if !poly.IsSolid {
				t.Errorf("Polygon %d lost IsSolid flag", i)
			}
		}
	})

	t.Run("Invalid polygon", func(t *testing.T) {
		// Too few vertices
		invalid := Polygon{
			Vertices: []Point{
				{X: 0, Y: 0},
				{X: 1, Y: 0},
			},
			IsSolid: true,
		}

		_, err := PartitionPolygonConvex(invalid)
		if err == nil {
			t.Error("Expected error for invalid polygon, got nil")
		}
	})
}
