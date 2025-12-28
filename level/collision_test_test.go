package level

import (
	"testing"

	"github.com/bloodmagesoftware/venture/bsp"
)

func TestBuildCollisionBSP(t *testing.T) {
	// Create a test level with some collision polygons
	level := &Level{
		Collisions: []Polygon{
			{
				Outline: []Vec2{
					{X: -5, Y: -5},
					{X: 5, Y: -5},
					{X: 5, Y: 5},
					{X: -5, Y: 5},
				},
			},
		},
	}

	// Create an editor with this level
	editor := &Editor{
		level:                 level,
		collisionTestBSPDirty: true,
	}

	// Build the BSP tree
	editor.buildCollisionBSP()

	// Verify BSP was built
	if editor.collisionTestBSP == nil {
		t.Fatal("BSP tree should not be nil after building")
	}

	// Verify BSP is no longer dirty
	if editor.collisionTestBSPDirty {
		t.Error("BSP tree should not be dirty after building")
	}

	// Test point inside polygon
	point := bsp.Point{X: 0, Y: 0}
	if !bsp.PointInBSP(editor.collisionTestBSP, point) {
		t.Error("Point (0, 0) should be inside the solid polygon")
	}

	// Test point outside polygon
	point = bsp.Point{X: 10, Y: 10}
	if bsp.PointInBSP(editor.collisionTestBSP, point) {
		t.Error("Point (10, 10) should be outside the solid polygon")
	}
}

func TestMarkCollisionBSPDirty(t *testing.T) {
	editor := &Editor{
		collisionTestBSPDirty: false,
	}

	editor.markCollisionBSPDirty()

	if !editor.collisionTestBSPDirty {
		t.Error("BSP should be marked as dirty after calling markCollisionBSPDirty")
	}
}

func TestLineTraceBSP(t *testing.T) {
	// Create a test level with a box in the center
	level := &Level{
		Collisions: []Polygon{
			{
				Outline: []Vec2{
					{X: -5, Y: -5},
					{X: 5, Y: -5},
					{X: 5, Y: 5},
					{X: -5, Y: 5},
				},
			},
		},
	}

	editor := &Editor{
		level:                 level,
		collisionTestBSPDirty: true,
	}

	// Build the BSP tree
	editor.buildCollisionBSP()

	// Test 1: Line from outside to inside should hit
	hit, hitX, _ := editor.lineTraceBSP(-10, 0, 0, 0)
	if !hit {
		t.Error("Line from (-10, 0) to (0, 0) should hit the solid box")
	}
	// Hit point should be at the edge of the box (x = -5)
	if hitX > -4.9 || hitX < -5.1 {
		t.Errorf("Hit X coordinate should be around -5, got %.3f", hitX)
	}

	// Test 2: Line entirely outside should not hit
	hit, _, _ = editor.lineTraceBSP(-20, -20, -10, -20)
	if hit {
		t.Error("Line from (-20, -20) to (-10, -20) should not hit (outside box)")
	}

	// Test 3: Line entirely inside should hit immediately (at start point)
	hit, hitX, _ = editor.lineTraceBSP(-2, 0, 2, 0)
	if !hit {
		t.Error("Line from (-2, 0) to (2, 0) should hit (inside solid)")
	}
	// Should hit at the start point
	if hitX > -1.9 || hitX < -2.1 {
		t.Errorf("Hit X coordinate should be around -2 (start point), got %.3f", hitX)
	}
}

func TestCollisionTestResult(t *testing.T) {
	// Test creating collision test results with line trace info
	result := collisionTestResult{
		WorldX:      1.0,
		WorldY:      2.0,
		IsSolid:     true,
		HasPrevious: true,
		PrevWorldX:  0.0,
		PrevWorldY:  1.0,
		LineHit:     true,
		LineHitX:    0.5,
		LineHitY:    1.5,
	}

	if result.WorldX != 1.0 {
		t.Errorf("Expected WorldX=1.0, got %.3f", result.WorldX)
	}
	if !result.IsSolid {
		t.Error("Expected IsSolid=true")
	}
	if !result.HasPrevious {
		t.Error("Expected HasPrevious=true")
	}
	if !result.LineHit {
		t.Error("Expected LineHit=true")
	}
}

