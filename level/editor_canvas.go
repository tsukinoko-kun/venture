//go:build !cli

package level

import (
	"image"
	"image/color"
	"log"
	"math"

	"github.com/bloodmagesoftware/venture/bsp"
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// layoutCanvas renders the main canvas area where level editing happens
func (e *Editor) layoutCanvas(gtx layout.Context) layout.Dimensions {
	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: color.NRGBA{R: 60, G: 60, B: 60, A: 255}}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
		func(gtx layout.Context) layout.Dimensions {
			// Handle pointer input for panning and zooming
			e.handleCanvasInput(gtx)

			// Draw the ground editor grid
			e.drawGroundGrid(gtx)
			// Draw collision polygons
			e.drawCollisionPolygons(gtx)
			// Draw collision test results (if collision test tool is active)
			if e.currentTool == "collision_test" {
				e.drawCollisionTestResults(gtx)
			}
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
	)
}

// handleCanvasInput processes mouse/pointer events for panning and zooming
func (e *Editor) handleCanvasInput(gtx layout.Context) {
	// Register for pointer input events
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)

	// Declare event handler for this canvas area
	event.Op(gtx.Ops, e)

	area.Pop()

	// Process all pointer events with scroll support
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target:  e,
			Kinds:   pointer.Press | pointer.Release | pointer.Drag | pointer.Scroll,
			ScrollY: pointer.ScrollRange{Min: -100, Max: 100},
		})
		if !ok {
			break
		}

		switch ev := ev.(type) {
		case pointer.Event:
			// Check if the pointer is within the canvas bounds
			canvasWidth := float32(gtx.Constraints.Max.X)
			canvasHeight := float32(gtx.Constraints.Max.Y)

			// Ignore events outside canvas bounds
			if ev.Position.X < 0 || ev.Position.X > canvasWidth || ev.Position.Y < 0 || ev.Position.Y > canvasHeight {
				continue
			}

			switch ev.Kind {
			case pointer.Press:
				// Start panning on right mouse button press
				if ev.Buttons == pointer.ButtonSecondary {
					e.isPanning = true
					e.lastMouseX = ev.Position.X
					e.lastMouseY = ev.Position.Y
				}
				// Place or delete tile on left mouse button press
				if ev.Buttons == pointer.ButtonPrimary {
					if e.currentTool == "ground" {
						if e.isDeleting {
							e.deleteTileAtPosition(gtx, ev.Position.X, ev.Position.Y)
						} else {
							e.placeTileAtPosition(gtx, ev.Position.X, ev.Position.Y)
						}
					} else if e.currentTool == "collision" {
						// If deleting mode is active
						if e.isDeleting {
							// If no polygon is selected, delete the entire polygon we clicked inside
							if e.selectedPolygonIndex < 0 {
								e.deletePolygonAtPosition(gtx, ev.Position.X, ev.Position.Y)
							} else {
								// Otherwise delete the nearest point in the selected polygon
								e.deleteCollisionPoint(gtx, ev.Position.X, ev.Position.Y)
							}
						} else if e.isMoving {
							// If moving mode is active, find the nearest point
							e.startMovingPoint(gtx, ev.Position.X, ev.Position.Y)
						} else if e.selectedPolygonIndex < 0 {
							// If no polygon is selected, try to select one by clicking inside it
							e.selectPolygonAtPosition(gtx, ev.Position.X, ev.Position.Y)
						} else {
							// Otherwise add a point to the selected polygon
							e.addCollisionPoint(gtx, ev.Position.X, ev.Position.Y)
						}
					} else if e.currentTool == "collision_test" {
						// Collision test tool: test point collision
						e.handleCollisionTest(gtx, ev.Position.X, ev.Position.Y)
					}
				}

			case pointer.Release:
				// Stop panning on right mouse button release
				if ev.Buttons&pointer.ButtonSecondary == 0 {
					e.isPanning = false
				}
				// Stop moving point on left mouse button release
				if ev.Buttons&pointer.ButtonPrimary == 0 {
					e.movingPointPolygonIndex = -1
					e.movingPointIndex = -1
				}

			case pointer.Drag:
				// Pan the canvas if right mouse button is held
				if e.isPanning {
					deltaX := ev.Position.X - e.lastMouseX
					deltaY := ev.Position.Y - e.lastMouseY
					e.viewOffsetX += deltaX
					e.viewOffsetY += deltaY
					e.lastMouseX = ev.Position.X
					e.lastMouseY = ev.Position.Y
				}
				// Place or delete tile while dragging with left mouse button
				if ev.Buttons == pointer.ButtonPrimary {
					if e.currentTool == "ground" {
						if e.isDeleting {
							e.deleteTileAtPosition(gtx, ev.Position.X, ev.Position.Y)
						} else {
							e.placeTileAtPosition(gtx, ev.Position.X, ev.Position.Y)
						}
					} else if e.currentTool == "collision" {
						// If moving a point, update its position
						if e.isMoving && e.movingPointPolygonIndex >= 0 && e.movingPointIndex >= 0 {
							e.movePoint(gtx, ev.Position.X, ev.Position.Y)
						}
					}
					// Note: We don't add collision points on drag, only on click
				}

			case pointer.Scroll:
				// Zoom in/out with mouse wheel
				// Scroll.Y is positive when scrolling up (zoom in), negative when scrolling down (zoom out)
				zoomFactor := float32(1.0 + ev.Scroll.Y*0.1)
				newZoom := e.zoom * zoomFactor

				// Clamp zoom to reasonable limits
				const minZoom = 0.1
				const maxZoom = 10.0
				if newZoom < minZoom {
					newZoom = minZoom
				}
				if newZoom > maxZoom {
					newZoom = maxZoom
				}

				// Zoom towards mouse position
				// Calculate the world position under the mouse before zoom
				canvasWidth := float32(gtx.Constraints.Max.X)
				canvasHeight := float32(gtx.Constraints.Max.Y)
				centerX := canvasWidth / 2.0
				centerY := canvasHeight / 2.0

				// Mouse position relative to center
				mouseRelX := ev.Position.X - centerX
				mouseRelY := ev.Position.Y - centerY

				// Adjust offset to keep the same world point under the mouse
				zoomRatio := newZoom / e.zoom
				e.viewOffsetX = (e.viewOffsetX-mouseRelX)*zoomRatio + mouseRelX
				e.viewOffsetY = (e.viewOffsetY-mouseRelY)*zoomRatio + mouseRelY

				e.zoom = newZoom
			}
		}
	}
}

// placeTileAtPosition places the selected texture at the grid position under the mouse
func (e *Editor) placeTileAtPosition(gtx layout.Context, mouseX, mouseY float32) {
	// Only place if we have a texture selected
	if e.selectedTexture == "" {
		return
	}

	// Convert screen coordinates to grid coordinates
	canvasWidth := float32(gtx.Constraints.Max.X)
	canvasHeight := float32(gtx.Constraints.Max.Y)
	centerX := canvasWidth / 2.0
	centerY := canvasHeight / 2.0

	cellSize := e.gridCellSize * e.zoom

	// Calculate grid position
	worldX := mouseX - centerX - e.viewOffsetX
	worldY := mouseY - centerY - e.viewOffsetY

	gridX := int32(math.Floor(float64(worldX / cellSize)))
	gridY := int32(math.Floor(float64(worldY / cellSize)))

	// Check if a tile already exists at this position
	tileIndex := -1
	for i, tile := range e.level.Ground {
		if tile.Position.X == gridX && tile.Position.Y == gridY {
			tileIndex = i
			break
		}
	}

	// Update or add the tile
	newTile := Tile{
		Position: Vec2i{X: gridX, Y: gridY},
		Texture:  e.selectedTexture,
	}

	if tileIndex >= 0 {
		// Update existing tile (only if different)
		if e.level.Ground[tileIndex].Texture != newTile.Texture {
			e.level.Ground[tileIndex] = newTile
			e.dirty = true // Mark as dirty when tile changes
		}
	} else {
		// Add new tile
		e.level.Ground = append(e.level.Ground, newTile)
		e.dirty = true // Mark as dirty when tile is added
	}
}

// deleteTileAtPosition removes the tile at the grid position under the mouse
func (e *Editor) deleteTileAtPosition(gtx layout.Context, mouseX, mouseY float32) {
	// Convert screen coordinates to grid coordinates
	canvasWidth := float32(gtx.Constraints.Max.X)
	canvasHeight := float32(gtx.Constraints.Max.Y)
	centerX := canvasWidth / 2.0
	centerY := canvasHeight / 2.0

	cellSize := e.gridCellSize * e.zoom

	// Calculate grid position
	worldX := mouseX - centerX - e.viewOffsetX
	worldY := mouseY - centerY - e.viewOffsetY

	gridX := int32(math.Floor(float64(worldX / cellSize)))
	gridY := int32(math.Floor(float64(worldY / cellSize)))

	// Find and remove the tile at this position
	for i, tile := range e.level.Ground {
		if tile.Position.X == gridX && tile.Position.Y == gridY {
			// Remove tile by replacing it with the last element and truncating
			e.level.Ground[i] = e.level.Ground[len(e.level.Ground)-1]
			e.level.Ground = e.level.Ground[:len(e.level.Ground)-1]
			e.dirty = true // Mark as dirty when tile is deleted
			return
		}
	}
}

// drawGroundGrid draws the placed tiles for the ground editor
func (e *Editor) drawGroundGrid(gtx layout.Context) {
	canvasWidth := float32(gtx.Constraints.Max.X)
	canvasHeight := float32(gtx.Constraints.Max.Y)

	// Clip all drawing operations to the canvas bounds
	defer clip.Rect{Max: image.Point{X: int(canvasWidth), Y: int(canvasHeight)}}.Push(gtx.Ops).Pop()

	// Calculate the center of the canvas (this will be our origin)
	centerX := canvasWidth / 2.0
	centerY := canvasHeight / 2.0

	// Apply zoom and offset to the cell size
	cellSize := e.gridCellSize * e.zoom

	// Draw placed tiles
	e.drawPlacedTiles(gtx, centerX, centerY, cellSize)
}

// drawPlacedTiles renders all tiles that have been placed in the level
func (e *Editor) drawPlacedTiles(gtx layout.Context, centerX, centerY, cellSize float32) {
	for _, tile := range e.level.Ground {
		// Calculate screen position for this tile
		screenX := centerX + e.viewOffsetX + float32(tile.Position.X)*cellSize
		screenY := centerY + e.viewOffsetY + float32(tile.Position.Y)*cellSize

		// Only draw tiles that are visible on screen
		canvasWidth := float32(gtx.Constraints.Max.X)
		canvasHeight := float32(gtx.Constraints.Max.Y)
		if screenX+cellSize < 0 || screenX > canvasWidth || screenY+cellSize < 0 || screenY > canvasHeight {
			continue
		}

		// Load the texture image
		img, err := e.loadAssetImage(tile.Texture)
		if err != nil {
			// Draw a placeholder rectangle if texture fails to load
			defer clip.Rect{
				Min: image.Point{X: int(screenX), Y: int(screenY)},
				Max: image.Point{X: int(screenX + cellSize), Y: int(screenY + cellSize)},
			}.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: color.NRGBA{R: 255, G: 0, B: 0, A: 128}}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			continue
		}

		// Save the current transform state
		stack := op.Affine(f32.Affine2D{}.Offset(f32.Point{X: screenX, Y: screenY})).Push(gtx.Ops)

		// Calculate scale to fit the texture in the cell
		imgBounds := img.Bounds()
		imgWidth := float32(imgBounds.Dx())
		imgHeight := float32(imgBounds.Dy())

		scaleX := cellSize / imgWidth
		scaleY := cellSize / imgHeight

		// Apply scaling transform
		scaleOp := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Point{X: scaleX, Y: scaleY})).Push(gtx.Ops)

		// Draw the image at the scaled size
		imageOp := paint.NewImageOp(img)
		imageOp.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)

		scaleOp.Pop()
		stack.Pop()
	}
}

// addCollisionPoint adds a new point to the currently selected collision polygon
func (e *Editor) addCollisionPoint(gtx layout.Context, mouseX, mouseY float32) {
	// Check if a polygon is selected
	if e.selectedPolygonIndex < 0 || e.selectedPolygonIndex >= len(e.level.Collisions) {
		// No valid polygon selected, do nothing
		return
	}

	// Convert screen coordinates to world coordinates
	canvasWidth := float32(gtx.Constraints.Max.X)
	canvasHeight := float32(gtx.Constraints.Max.Y)
	centerX := canvasWidth / 2.0
	centerY := canvasHeight / 2.0

	// Calculate world position (1 unit = 1 tile = gridCellSize pixels)
	cellSize := e.gridCellSize * e.zoom
	worldX := (mouseX - centerX - e.viewOffsetX) / cellSize
	worldY := (mouseY - centerY - e.viewOffsetY) / cellSize

	// Snap to 0.125 grid
	originalX := worldX
	originalY := worldY
	worldX = snapToGrid(worldX, 0.125)
	worldY = snapToGrid(worldY, 0.125)

	log.Printf("Adding point: original=(%.6f, %.6f), snapped=(%.6f, %.6f)", originalX, originalY, worldX, worldY)

	// Add the point to the selected collision polygon
	point := Vec2{X: worldX, Y: worldY}
	e.level.Collisions[e.selectedPolygonIndex].Outline = append(e.level.Collisions[e.selectedPolygonIndex].Outline, point)
	e.dirty = true
	e.markCollisionBSPDirty() // Mark BSP as needing rebuild
}

// snapToGrid snaps a coordinate to the nearest grid point
func snapToGrid(value float32, gridSize float32) float32 {
	return float32(math.Round(float64(value/gridSize))) * gridSize
}

// startMovingPoint finds the nearest point to the mouse in the selected polygon and starts moving it
func (e *Editor) startMovingPoint(gtx layout.Context, mouseX, mouseY float32) {
	// Check if a polygon is selected
	if e.selectedPolygonIndex < 0 || e.selectedPolygonIndex >= len(e.level.Collisions) {
		// No valid polygon selected, do nothing
		return
	}

	canvasWidth := float32(gtx.Constraints.Max.X)
	canvasHeight := float32(gtx.Constraints.Max.Y)
	centerX := canvasWidth / 2.0
	centerY := canvasHeight / 2.0
	cellSize := e.gridCellSize * e.zoom

	// Find the nearest point within a threshold distance (only in selected polygon)
	const clickThreshold = 15.0 // pixels
	minDist := float32(clickThreshold)
	foundPointIndex := -1

	polygon := e.level.Collisions[e.selectedPolygonIndex]
	for pointIdx, point := range polygon.Outline {
		// Convert world coordinates to screen coordinates (1 unit = 1 tile)
		screenX := centerX + e.viewOffsetX + point.X*cellSize
		screenY := centerY + e.viewOffsetY + point.Y*cellSize

		// Calculate distance from mouse to point
		dx := screenX - mouseX
		dy := screenY - mouseY
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))

		if dist < minDist {
			minDist = dist
			foundPointIndex = pointIdx
		}
	}

	// If we found a point, start moving it
	if foundPointIndex >= 0 {
		e.movingPointPolygonIndex = e.selectedPolygonIndex
		e.movingPointIndex = foundPointIndex
	}
}

// movePoint updates the position of the currently moving point
func (e *Editor) movePoint(gtx layout.Context, mouseX, mouseY float32) {
	// Check if we have a valid point to move
	if e.movingPointPolygonIndex < 0 || e.movingPointPolygonIndex >= len(e.level.Collisions) {
		return
	}
	if e.movingPointIndex < 0 || e.movingPointIndex >= len(e.level.Collisions[e.movingPointPolygonIndex].Outline) {
		return
	}

	// Convert screen coordinates to world coordinates (1 unit = 1 tile)
	canvasWidth := float32(gtx.Constraints.Max.X)
	canvasHeight := float32(gtx.Constraints.Max.Y)
	centerX := canvasWidth / 2.0
	centerY := canvasHeight / 2.0
	cellSize := e.gridCellSize * e.zoom

	worldX := (mouseX - centerX - e.viewOffsetX) / cellSize
	worldY := (mouseY - centerY - e.viewOffsetY) / cellSize

	// Snap to 0.125 grid
	worldX = snapToGrid(worldX, 0.125)
	worldY = snapToGrid(worldY, 0.125)

	// Update the point position
	e.level.Collisions[e.movingPointPolygonIndex].Outline[e.movingPointIndex] = Vec2{X: worldX, Y: worldY}
	e.dirty = true
	e.markCollisionBSPDirty() // Mark BSP as needing rebuild
}

// deleteCollisionPoint finds and deletes the nearest collision point to the mouse in the selected polygon
func (e *Editor) deleteCollisionPoint(gtx layout.Context, mouseX, mouseY float32) {
	// Check if a polygon is selected
	if e.selectedPolygonIndex < 0 || e.selectedPolygonIndex >= len(e.level.Collisions) {
		// No valid polygon selected, do nothing
		return
	}

	canvasWidth := float32(gtx.Constraints.Max.X)
	canvasHeight := float32(gtx.Constraints.Max.Y)
	centerX := canvasWidth / 2.0
	centerY := canvasHeight / 2.0
	cellSize := e.gridCellSize * e.zoom

	// Find the nearest point within a threshold distance (only in selected polygon)
	const clickThreshold = 15.0 // pixels
	minDist := float32(clickThreshold)
	foundPointIndex := -1

	polygon := e.level.Collisions[e.selectedPolygonIndex]
	for pointIdx, point := range polygon.Outline {
		// Convert world coordinates to screen coordinates (1 unit = 1 tile)
		screenX := centerX + e.viewOffsetX + point.X*cellSize
		screenY := centerY + e.viewOffsetY + point.Y*cellSize

		// Calculate distance from mouse to point
		dx := screenX - mouseX
		dy := screenY - mouseY
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))

		if dist < minDist {
			minDist = dist
			foundPointIndex = pointIdx
		}
	}

	// If we found a point, delete it
	if foundPointIndex >= 0 {
		polygon := &e.level.Collisions[e.selectedPolygonIndex]

		// Remove the point by slicing
		polygon.Outline = append(polygon.Outline[:foundPointIndex], polygon.Outline[foundPointIndex+1:]...)

		// If the polygon is now empty, remove it
		if len(polygon.Outline) == 0 {
			e.level.Collisions = append(e.level.Collisions[:e.selectedPolygonIndex], e.level.Collisions[e.selectedPolygonIndex+1:]...)
			// Clear the selection since the polygon was deleted
			e.selectedPolygonIndex = -1
		}

		e.dirty = true
		e.markCollisionBSPDirty() // Mark BSP as needing rebuild
		log.Printf("Deleted collision point at polygon %d, point %d", e.selectedPolygonIndex, foundPointIndex)
	}
}

// drawCollisionPolygons draws all collision polygons and their points
func (e *Editor) drawCollisionPolygons(gtx layout.Context) {
	canvasWidth := float32(gtx.Constraints.Max.X)
	canvasHeight := float32(gtx.Constraints.Max.Y)

	// Clip all drawing operations to the canvas bounds
	defer clip.Rect{Max: image.Point{X: int(canvasWidth), Y: int(canvasHeight)}}.Push(gtx.Ops).Pop()

	// Calculate the center of the canvas
	centerX := canvasWidth / 2.0
	centerY := canvasHeight / 2.0

	// Draw all collision polygons
	for _, polygon := range e.level.Collisions {
		e.drawCollisionPolygon(gtx, polygon, centerX, centerY)
	}
}

// drawCollisionPolygon draws a single collision polygon and its points
func (e *Editor) drawCollisionPolygon(gtx layout.Context, polygon Polygon, centerX, centerY float32) {
	cellSize := e.gridCellSize * e.zoom

	// Fill the polygon if we have at least 3 points
	if len(polygon.Outline) >= 3 {
		var path clip.Path
		path.Begin(gtx.Ops)

		// Move to the first point
		p0 := polygon.Outline[0]
		screenX := centerX + e.viewOffsetX + p0.X*cellSize
		screenY := centerY + e.viewOffsetY + p0.Y*cellSize
		path.MoveTo(f32.Point{X: screenX, Y: screenY})

		// Draw lines to all other points
		for i := 1; i < len(polygon.Outline); i++ {
			p := polygon.Outline[i]
			screenX := centerX + e.viewOffsetX + p.X*cellSize
			screenY := centerY + e.viewOffsetY + p.Y*cellSize
			path.LineTo(f32.Point{X: screenX, Y: screenY})
		}

		// Close the path back to the first point
		path.Close()

		// Fill the polygon with semi-transparent cyan
		spec := path.End()
		stack := clip.Outline{Path: spec}.Op().Push(gtx.Ops)
		paint.ColorOp{Color: color.NRGBA{R: 100, G: 200, B: 255, A: 60}}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		stack.Pop() // Pop immediately so subsequent drawing isn't clipped
	}

	// Draw lines between consecutive points
	if len(polygon.Outline) > 1 {
		for i := 0; i < len(polygon.Outline)-1; i++ {
			p1 := polygon.Outline[i]
			p2 := polygon.Outline[i+1]

			// Convert world coordinates to screen coordinates (1 unit = 1 tile)
			screenX1 := centerX + e.viewOffsetX + p1.X*cellSize
			screenY1 := centerY + e.viewOffsetY + p1.Y*cellSize
			screenX2 := centerX + e.viewOffsetX + p2.X*cellSize
			screenY2 := centerY + e.viewOffsetY + p2.Y*cellSize

			// Draw a line between the two points
			e.drawLine(gtx, screenX1, screenY1, screenX2, screenY2, 2.0, color.NRGBA{R: 100, G: 200, B: 255, A: 200})
		}

		// Draw a line from the last point back to the first point to close the polygon
		pFirst := polygon.Outline[0]
		pLast := polygon.Outline[len(polygon.Outline)-1]

		screenX1 := centerX + e.viewOffsetX + pLast.X*cellSize
		screenY1 := centerY + e.viewOffsetY + pLast.Y*cellSize
		screenX2 := centerX + e.viewOffsetX + pFirst.X*cellSize
		screenY2 := centerY + e.viewOffsetY + pFirst.Y*cellSize

		e.drawLine(gtx, screenX1, screenY1, screenX2, screenY2, 2.0, color.NRGBA{R: 100, G: 200, B: 255, A: 200})
	}

	// Draw all points as circles (on top of lines)
	for _, point := range polygon.Outline {
		// Convert world coordinates to screen coordinates (1 unit = 1 tile)
		screenX := centerX + e.viewOffsetX + point.X*cellSize
		screenY := centerY + e.viewOffsetY + point.Y*cellSize

		// Draw a circle at this point
		e.drawCircle(gtx, screenX, screenY, 6.0, color.NRGBA{R: 100, G: 200, B: 255, A: 255})
	}
}

// drawCircle draws a filled circle at the given position
func (e *Editor) drawCircle(gtx layout.Context, x, y, radius float32, col color.NRGBA) {
	// Create a circular path
	const segments = 32
	var path clip.Path
	path.Begin(gtx.Ops)

	// Start at the rightmost point of the circle
	firstX := x + radius
	firstY := y
	path.MoveTo(f32.Point{X: firstX, Y: firstY})

	// Draw the circle using line segments
	for i := 1; i <= segments; i++ {
		angle := float32(i) * 2.0 * math.Pi / segments
		px := x + radius*float32(math.Cos(float64(angle)))
		py := y + radius*float32(math.Sin(float64(angle)))
		path.LineTo(f32.Point{X: px, Y: py})
	}

	path.Close()

	// Fill the circle
	spec := path.End()
	defer clip.Outline{Path: spec}.Op().Push(gtx.Ops).Pop()
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

// drawLine draws a line between two points with the given width
func (e *Editor) drawLine(gtx layout.Context, x1, y1, x2, y2, width float32, col color.NRGBA) {
	// Create a stroked path for the line
	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(f32.Point{X: x1, Y: y1})
	path.LineTo(f32.Point{X: x2, Y: y2})

	// Stroke the path with the given width
	spec := path.End()
	stroke := clip.Stroke{
		Path:  spec,
		Width: width,
	}.Op()

	defer stroke.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

// selectPolygonAtPosition selects a polygon if the click position is inside it
func (e *Editor) selectPolygonAtPosition(gtx layout.Context, mouseX, mouseY float32) {
	// Convert screen coordinates to world coordinates
	canvasWidth := float32(gtx.Constraints.Max.X)
	canvasHeight := float32(gtx.Constraints.Max.Y)
	centerX := canvasWidth / 2.0
	centerY := canvasHeight / 2.0
	cellSize := e.gridCellSize * e.zoom

	worldX := (mouseX - centerX - e.viewOffsetX) / cellSize
	worldY := (mouseY - centerY - e.viewOffsetY) / cellSize

	// Check each polygon to see if the point is inside
	// Check in reverse order so topmost polygons are selected first
	for i := len(e.level.Collisions) - 1; i >= 0; i-- {
		polygon := e.level.Collisions[i]
		if len(polygon.Outline) >= 3 && isPointInPolygon(worldX, worldY, polygon.Outline) {
			e.selectedPolygonIndex = i
			log.Printf("Selected polygon %d", i)
			return
		}
	}
}

// isPointInPolygon uses the ray casting algorithm to determine if a point is inside a polygon
func isPointInPolygon(x, y float32, polygon []Vec2) bool {
	if len(polygon) < 3 {
		return false
	}

	inside := false
	j := len(polygon) - 1

	for i := 0; i < len(polygon); i++ {
		xi, yi := polygon[i].X, polygon[i].Y
		xj, yj := polygon[j].X, polygon[j].Y

		// Ray casting algorithm
		if ((yi > y) != (yj > y)) && (x < (xj-xi)*(y-yi)/(yj-yi)+xi) {
			inside = !inside
		}
		j = i
	}

	return inside
}

// deletePolygonAtPosition deletes an entire polygon if the click position is inside it
func (e *Editor) deletePolygonAtPosition(gtx layout.Context, mouseX, mouseY float32) {
	// Convert screen coordinates to world coordinates
	canvasWidth := float32(gtx.Constraints.Max.X)
	canvasHeight := float32(gtx.Constraints.Max.Y)
	centerX := canvasWidth / 2.0
	centerY := canvasHeight / 2.0
	cellSize := e.gridCellSize * e.zoom

	worldX := (mouseX - centerX - e.viewOffsetX) / cellSize
	worldY := (mouseY - centerY - e.viewOffsetY) / cellSize

	// Check each polygon to see if the point is inside
	// Check in reverse order so topmost polygons are deleted first
	for i := len(e.level.Collisions) - 1; i >= 0; i-- {
		polygon := e.level.Collisions[i]
		if len(polygon.Outline) >= 3 && isPointInPolygon(worldX, worldY, polygon.Outline) {
			// Delete the polygon
			e.level.Collisions = append(e.level.Collisions[:i], e.level.Collisions[i+1:]...)
			e.dirty = true
			e.markCollisionBSPDirty() // Mark BSP as needing rebuild
			log.Printf("Deleted entire polygon %d", i)
			return
		}
	}
}

// handleCollisionTest handles click for collision test tool
func (e *Editor) handleCollisionTest(gtx layout.Context, mouseX, mouseY float32) {
	// Convert screen coordinates to world coordinates
	canvasWidth := float32(gtx.Constraints.Max.X)
	canvasHeight := float32(gtx.Constraints.Max.Y)
	centerX := canvasWidth / 2.0
	centerY := canvasHeight / 2.0
	cellSize := e.gridCellSize * e.zoom

	worldX := (mouseX - centerX - e.viewOffsetX) / cellSize
	worldY := (mouseY - centerY - e.viewOffsetY) / cellSize

	// Rebuild BSP if dirty
	if e.collisionTestBSPDirty || e.collisionTestBSP == nil {
		e.buildCollisionBSP()
	}

	// Test point collision
	point := bsp.Point{X: worldX, Y: worldY}
	isSolid := bsp.PointInBSP(e.collisionTestBSP, point)

	// Create result with point test
	result := collisionTestResult{
		WorldX:  worldX,
		WorldY:  worldY,
		IsSolid: isSolid,
	}

	// If we have a previous point, do line trace
	if len(e.collisionTestPoints) > 0 {
		prev := e.collisionTestPoints[len(e.collisionTestPoints)-1]
		result.HasPrevious = true
		result.PrevWorldX = prev.WorldX
		result.PrevWorldY = prev.WorldY

		// Perform line trace from previous point to current point
		hit, hitX, hitY := e.lineTraceBSP(prev.WorldX, prev.WorldY, worldX, worldY)
		result.LineHit = hit
		result.LineHitX = hitX
		result.LineHitY = hitY

		log.Printf("Line trace from (%.3f, %.3f) to (%.3f, %.3f): hit=%v", 
			prev.WorldX, prev.WorldY, worldX, worldY, hit)
		if hit {
			log.Printf("  Hit at (%.3f, %.3f)", hitX, hitY)
		}
	}

	// Store result
	e.collisionTestPoints = append(e.collisionTestPoints, result)

	log.Printf("Collision test at (%.3f, %.3f): solid=%v", worldX, worldY, isSolid)
}

// drawCollisionTestResults renders the collision test results on the canvas
func (e *Editor) drawCollisionTestResults(gtx layout.Context) {
	if len(e.collisionTestPoints) == 0 {
		return
	}

	canvasWidth := float32(gtx.Constraints.Max.X)
	canvasHeight := float32(gtx.Constraints.Max.Y)
	centerX := canvasWidth / 2.0
	centerY := canvasHeight / 2.0
	cellSize := e.gridCellSize * e.zoom

	// Helper to convert world coords to screen coords
	worldToScreen := func(worldX, worldY float32) (float32, float32) {
		screenX := worldX*cellSize + centerX + e.viewOffsetX
		screenY := worldY*cellSize + centerY + e.viewOffsetY
		return screenX, screenY
	}

	// Colors from the plan
	colorGreen := color.NRGBA{R: 0, G: 200, B: 80, A: 255}   // Point (non-solid)
	colorRed := color.NRGBA{R: 255, G: 60, B: 60, A: 255}     // Point (solid)
	colorPink := color.NRGBA{R: 255, G: 100, B: 180, A: 255}  // Line trace & intersection

	// Circle radius in pixels
	circleRadius := float32(6.0)

	// Draw each test result
	for _, result := range e.collisionTestPoints {
		screenX, screenY := worldToScreen(result.WorldX, result.WorldY)

		// Draw line trace if it has a previous point
		if result.HasPrevious {
			prevScreenX, prevScreenY := worldToScreen(result.PrevWorldX, result.PrevWorldY)
			
			// Draw pink line from previous to current
			e.drawLine(gtx, prevScreenX, prevScreenY, screenX, screenY, 2.0, colorPink)

			// Draw intersection circle if line hit solid
			if result.LineHit {
				hitScreenX, hitScreenY := worldToScreen(result.LineHitX, result.LineHitY)
				e.drawCircle(gtx, hitScreenX, hitScreenY, circleRadius, colorPink)
			}
		}

		// Draw point circle (green if non-solid, red if solid)
		pointColor := colorGreen
		if result.IsSolid {
			pointColor = colorRed
		}
		e.drawCircle(gtx, screenX, screenY, circleRadius, pointColor)
	}
}
