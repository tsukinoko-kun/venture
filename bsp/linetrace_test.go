package bsp

import (
	"math"
	"testing"

	pb "github.com/bloodmagesoftware/venture/proto/level"
)

// LineTraceResult holds the result of a line trace
type LineTraceResult struct {
	Hit  bool
	HitX float32
	HitY float32
}

// LineTraceBSP traces a line through a BSP tree and returns the first hit point
func LineTraceBSP(root *pb.BSPNode, fromX, fromY, toX, toY float32) LineTraceResult {
	from := Point{X: fromX, Y: fromY}
	to := Point{X: toX, Y: toY}
	hit, hitX, hitY := LineTraceBSPNode(root, from, to, 0.0, 1.0)
	return LineTraceResult{Hit: hit, HitX: hitX, HitY: hitY}
}

func TestLineTraceSimpleBox(t *testing.T) {
	// Create a simple 10x10 box centered at origin
	poly := Polygon{
		Vertices: []Point{
			{X: -5, Y: -5},
			{X: 5, Y: -5},
			{X: 5, Y: 5},
			{X: -5, Y: 5},
		},
		IsSolid: true,
	}

	builder := NewBSPBuilder([]Polygon{poly})
	root := builder.Build()

	tests := []struct {
		name         string
		fromX, fromY float32
		toX, toY     float32
		expectHit    bool
		expectHitX   float32
		expectHitY   float32
		tolerance    float32
	}{
		// Lines from outside to inside
		{
			name:  "Horizontal line from left into box",
			fromX: -10, fromY: 0,
			toX: 0, toY: 0,
			expectHit:  true,
			expectHitX: -5, expectHitY: 0,
			tolerance: 0.1,
		},
		{
			name:  "Horizontal line from right into box",
			fromX: 10, fromY: 0,
			toX: 0, toY: 0,
			expectHit:  true,
			expectHitX: 5, expectHitY: 0,
			tolerance: 0.1,
		},
		{
			name:  "Vertical line from top into box",
			fromX: 0, fromY: 10,
			toX: 0, toY: 0,
			expectHit:  true,
			expectHitX: 0, expectHitY: 5,
			tolerance: 0.1,
		},
		{
			name:  "Vertical line from bottom into box",
			fromX: 0, fromY: -10,
			toX: 0, toY: 0,
			expectHit:  true,
			expectHitX: 0, expectHitY: -5,
			tolerance: 0.1,
		},
		{
			name:  "Diagonal line from corner into box",
			fromX: -10, fromY: -10,
			toX: 0, toY: 0,
			expectHit:  true,
			expectHitX: -5, expectHitY: -5,
			tolerance: 0.1,
		},
		// Lines entirely outside
		{
			name:  "Horizontal line missing box (above)",
			fromX: -10, fromY: 10,
			toX: 10, toY: 10,
			expectHit: false,
		},
		{
			name:  "Vertical line missing box (right)",
			fromX: 10, fromY: -10,
			toX: 10, toY: 10,
			expectHit: false,
		},
		// Lines entirely inside
		{
			name:  "Line starting inside",
			fromX: 0, fromY: 0,
			toX: 2, toY: 2,
			expectHit:  true,
			expectHitX: 0, expectHitY: 0, // Should hit at start
			tolerance: 0.1,
		},
		// Lines through the box
		{
			name:  "Line through box horizontally",
			fromX: -10, fromY: 0,
			toX: 10, toY: 0,
			expectHit:  true,
			expectHitX: -5, expectHitY: 0, // First hit at entry
			tolerance: 0.1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := LineTraceBSP(root, tc.fromX, tc.fromY, tc.toX, tc.toY)

			if result.Hit != tc.expectHit {
				t.Errorf("Expected hit=%v, got hit=%v", tc.expectHit, result.Hit)
				return
			}

			if tc.expectHit {
				dx := result.HitX - tc.expectHitX
				dy := result.HitY - tc.expectHitY
				dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
				if dist > tc.tolerance {
					t.Errorf("Hit point (%.2f, %.2f) too far from expected (%.2f, %.2f), distance=%.2f",
						result.HitX, result.HitY, tc.expectHitX, tc.expectHitY, dist)
				}
			}
		})
	}
}

func TestLineTraceMultiplePolygons(t *testing.T) {
	// Two boxes separated by a gap
	polygons := []Polygon{
		{
			Vertices: []Point{
				{X: -10, Y: -5},
				{X: -5, Y: -5},
				{X: -5, Y: 5},
				{X: -10, Y: 5},
			},
			IsSolid: true,
		},
		{
			Vertices: []Point{
				{X: 5, Y: -5},
				{X: 10, Y: -5},
				{X: 10, Y: 5},
				{X: 5, Y: 5},
			},
			IsSolid: true,
		},
	}

	builder := NewBSPBuilder(polygons)
	root := builder.Build()

	tests := []struct {
		name         string
		fromX, fromY float32
		toX, toY     float32
		expectHit    bool
		expectHitX   float32
		expectHitY   float32
		tolerance    float32
	}{
		{
			name:  "Line hitting left box",
			fromX: -15, fromY: 0,
			toX: 0, toY: 0,
			expectHit:  true,
			expectHitX: -10, expectHitY: 0,
			tolerance: 0.1,
		},
		{
			name:  "Line hitting right box",
			fromX: 0, fromY: 0,
			toX: 15, toY: 0,
			expectHit:  true,
			expectHitX: 5, expectHitY: 0,
			tolerance: 0.1,
		},
		{
			name:  "Line through gap (no hit)",
			fromX: 0, fromY: -10,
			toX: 0, toY: 10,
			expectHit: false,
		},
		{
			name:  "Line through both boxes (hits left first)",
			fromX: -15, fromY: 0,
			toX: 15, toY: 0,
			expectHit:  true,
			expectHitX: -10, expectHitY: 0,
			tolerance: 0.1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := LineTraceBSP(root, tc.fromX, tc.fromY, tc.toX, tc.toY)

			if result.Hit != tc.expectHit {
				t.Errorf("Expected hit=%v, got hit=%v", tc.expectHit, result.Hit)
				return
			}

			if tc.expectHit {
				dx := result.HitX - tc.expectHitX
				dy := result.HitY - tc.expectHitY
				dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
				if dist > tc.tolerance {
					t.Errorf("Hit point (%.2f, %.2f) too far from expected (%.2f, %.2f), distance=%.2f",
						result.HitX, result.HitY, tc.expectHitX, tc.expectHitY, dist)
				}
			}
		})
	}
}

func TestLineTraceLevelPolygons(t *testing.T) {
	// Use exact polygons from test.yaml level file
	poly2 := Polygon{
		Vertices: []Point{
			{X: -3, Y: -1},
			{X: -3, Y: 0},
			{X: 3, Y: 0},
			{X: 3, Y: -1},
		},
		IsSolid: true,
	}

	poly3 := Polygon{
		Vertices: []Point{
			{X: -3, Y: 2},
			{X: 3, Y: 2},
			{X: 3, Y: 0},
			{X: -3, Y: 0},
		},
		IsSolid: true,
	}

	builder := NewBSPBuilder([]Polygon{poly2, poly3})
	root := builder.Build()

	tests := []struct {
		name         string
		fromX, fromY float32
		toX, toY     float32
		expectHit    bool
		expectHitX   float32
		expectHitY   float32
		tolerance    float32
	}{
		{
			name:  "Line from below into poly2",
			fromX: 0, fromY: -5,
			toX: 0, toY: -0.5,
			expectHit:  true,
			expectHitX: 0, expectHitY: -1,
			tolerance: 0.1,
		},
		{
			name:  "Line from above into poly3",
			fromX: 0, fromY: 5,
			toX: 0, toY: 1,
			expectHit:  true,
			expectHitX: 0, expectHitY: 2,
			tolerance: 0.1,
		},
		{
			name:  "Line from left into poly2",
			fromX: -5, fromY: -0.5,
			toX: 0, toY: -0.5,
			expectHit:  true,
			expectHitX: -3, expectHitY: -0.5,
			tolerance: 0.1,
		},
		{
			name:  "Vertical line through both polys (hits poly2 first)",
			fromX: 0, fromY: -5,
			toX: 0, toY: 5,
			expectHit:  true,
			expectHitX: 0, expectHitY: -1,
			tolerance: 0.1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := LineTraceBSP(root, tc.fromX, tc.fromY, tc.toX, tc.toY)

			if result.Hit != tc.expectHit {
				t.Errorf("Expected hit=%v, got hit=%v", tc.expectHit, result.Hit)
				return
			}

			if tc.expectHit {
				dx := result.HitX - tc.expectHitX
				dy := result.HitY - tc.expectHitY
				dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
				if dist > tc.tolerance {
					t.Errorf("Hit point (%.2f, %.2f) too far from expected (%.2f, %.2f), distance=%.2f",
						result.HitX, result.HitY, tc.expectHitX, tc.expectHitY, dist)
				}
			}
		})
	}
}
