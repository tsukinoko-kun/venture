# Collision Test Tool Implementation

## Overview

The Collision Test Tool is a new feature in the level editor that allows interactive testing of BSP collision detection at click points and performs line traces between consecutive clicks. Visual feedback uses colored circles (green/red) and pink lines with intersection markers.

## Implementation Summary

### 1. Editor State Extensions

Added new fields to the `Editor` struct in `level/editor.go`:

- `collisionTestButton`: UI button for the collision test tool
- `collisionTestPoints`: History of test results
- `collisionTestBSP`: Cached BSP tree built from level collisions
- `collisionTestBSPDirty`: Flag indicating when BSP needs rebuilding

Created `collisionTestResult` struct to store:
- Point coordinates (world space)
- Collision result (solid/non-solid)
- Line trace information (previous point, hit status, hit coordinates)

### 2. Tool UI Integration

Modified `level/editor_ui.go`:
- Added "Collision Test" button to the tool list (3rd tool)
- Increased tool list count from 2 to 3
- Added button click handler that switches to `collision_test` mode
- Clears previous test results when tool is activated

### 3. BSP Building

Implemented in `level/editor.go`:

- `buildCollisionBSP()`: Converts level collision polygons to BSP format and builds the tree
- `markCollisionBSPDirty()`: Marks BSP as needing rebuild when collisions change
- Integrated dirty marking into all collision modification operations:
  - Adding collision points
  - Moving collision points
  - Deleting collision points
  - Deleting entire polygons
  - Creating new polygons

### 4. Click Handling

Implemented in `level/editor_canvas.go`:

- `handleCollisionTest()`: Main handler for collision test tool clicks
  - Converts screen coordinates to world coordinates
  - Rebuilds BSP if dirty
  - Tests point collision using `bsp.PointInBSP()`
  - Performs line trace if there's a previous point
  - Stores result with all relevant data

### 5. Line Trace Implementation

Implemented in `level/editor.go`:

- `lineTraceBSP()`: Public method that initiates line trace
- `lineTraceBSPNode()`: Recursive method that traces through BSP tree nodes
  - Handles leaf nodes (returns hit if solid)
  - Handles split nodes (classifies endpoints, splits line if needed)
  - Uses parametric line representation [t0, t1] for segment subdivision
  - Traverses near side first, then far side
  - Returns first solid hit found

Algorithm:
1. Classify both line endpoints against current split plane
2. If both on same side, recurse into that child
3. If spanning, calculate intersection point and recurse into both sides (near first)
4. Return first solid leaf hit

### 6. Visual Rendering

Implemented in `level/editor_canvas.go`:

- `drawCollisionTestResults()`: Main rendering method
  - Converts world coordinates to screen coordinates
  - Draws pink lines between consecutive test points
  - Draws pink circles at intersection points (if line hit solid)
  - Draws green circles for non-solid points
  - Draws red circles for solid points
- Uses existing `drawCircle()` and `drawLine()` helper methods

Colors (as specified in plan):
- Green (0, 200, 80): Non-solid points
- Red (255, 60, 60): Solid points
- Pink (255, 100, 180): Line traces and intersections

## Testing

Created comprehensive tests in `level/collision_test_test.go`:

1. **TestBuildCollisionBSP**: Verifies BSP tree building from level collisions
2. **TestMarkCollisionBSPDirty**: Verifies dirty flag management
3. **TestLineTraceBSP**: Tests line tracing through BSP tree
   - Line from outside to inside (should hit at boundary)
   - Line entirely outside (should not hit)
   - Line entirely inside (should hit at start point)
4. **TestCollisionTestResult**: Verifies result data structure

All tests pass successfully.

## Usage

1. Click the "Collision Test" button in the tool list (right panel)
2. Click anywhere on the canvas to test collision at that point
3. Continue clicking to perform line traces between consecutive points
4. Visual feedback:
   - Green circle: Point is in empty space (non-solid)
   - Red circle: Point is inside solid geometry
   - Pink line: Line trace from previous point
   - Pink circle on line: Line trace hit solid geometry at this point

## Integration Notes

- BSP tree is automatically rebuilt when collisions are modified
- Test results are cleared when switching to collision test tool
- Works with all collision polygons (convex and concave, via CGAL partitioning)
- BSP tree uses CGAL for convex polygon partitioning (see `bsp` package)

## Dependencies

- `bsp` package: Provides BSP tree construction and collision detection
- CGAL library: Used by BSP package for polygon partitioning (requires `DYLD_LIBRARY_PATH` on macOS)
- Gio UI: For rendering and input handling

## Files Modified

1. `level/editor.go`: Editor state, BSP building, line trace logic
2. `level/editor_ui.go`: Tool button UI
3. `level/editor_canvas.go`: Click handling, rendering
4. `level/collision_test_test.go`: Integration tests (new)

## Build and Run

Build:
```bash
cd /Users/frank/Git/bms/venture
go build
```

Run tests:
```bash
cd /Users/frank/Git/bms/venture
DYLD_LIBRARY_PATH="${DYLD_LIBRARY_PATH}:${PWD}/bsp/cgal" go test -v ./level -run TestCollision
```

Run editor:
```bash
./venture level test.yaml
```

## Future Enhancements

Possible improvements:
- Add keyboard shortcut to clear test results
- Add UI to show test statistics (hit count, miss count)
- Add option to export test results
- Add visual indicators for test point numbers/sequence
- Add option to toggle BSP tree visualization

