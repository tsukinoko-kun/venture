package level

import (
	"image"
	"log"

	"github.com/bloodmagesoftware/venture/bsp"
	pb "github.com/bloodmagesoftware/venture/proto/level"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

// collisionTestResult stores the result of a single collision test
type collisionTestResult struct {
	WorldX  float32 // world X coordinate
	WorldY  float32 // world Y coordinate
	IsSolid bool    // collision result
	// Line trace from previous point
	HasPrevious     bool    // true if there's a previous point to trace from
	PrevWorldX      float32 // previous point X coordinate
	PrevWorldY      float32 // previous point Y coordinate
	LineHit         bool    // true if line trace hit solid
	LineHitX        float32 // line trace hit X coordinate
	LineHitY        float32 // line trace hit Y coordinate
}

// Editor is the main level editor component that manages the UI state and interactions
type Editor struct {
	theme         *material.Theme
	levelFilePath string
	assetsDir     string
	level         *Level

	// UI state
	assetFiles      []string
	folderStructure map[string][]string // folder path -> list of files in that folder
	folders         []string            // list of all folders
	selectedFolder  string              // currently selected folder
	selectedTexture string              // currently selected texture path (for painting)
	toolList        widget.List
	assetList       widget.List
	folderList      widget.List
	folderButtons   []widget.Clickable     // clickable widgets for each folder
	assetImages     map[string]image.Image // cache of loaded QOI textures
	assetButtons    []widget.Clickable     // clickable widgets for each asset in grid
	saveButton      widget.Clickable       // button to save the level
	saveIcon        *widget.Icon           // icon for the save button
	dirty           bool                   // true when there are unsaved changes

	// Close confirmation dialog
	showCloseDialog    bool             // true when showing the close confirmation dialog
	closeSaveButton    widget.Clickable // "Save" button in close dialog
	closeDiscardButton widget.Clickable // "Discard" button in close dialog
	shouldClose        bool             // true when window should actually close

	// Canvas state
	gridCellSize float32 // size of one grid cell in screen pixels
	viewOffsetX  float32 // camera pan offset X
	viewOffsetY  float32 // camera pan offset Y
	zoom         float32 // zoom level (1.0 = 100%)

	// Mouse/pointer state for canvas interaction
	isPanning  bool    // true when right mouse button is held down
	lastMouseX float32 // last mouse X position for drag calculation
	lastMouseY float32 // last mouse Y position for drag calculation

	// Keyboard state for canvas interaction
	isDeleting bool // true when 'x' key is held down
	isMoving   bool // true when 'm' key is held down

	// Point moving state
	movingPointPolygonIndex int // index of the polygon being edited (-1 = none)
	movingPointIndex        int // index of the point being moved (-1 = none)

	// Tool state
	currentTool         string           // currently active tool ("ground", "collision", or "collision_test")
	groundButton        widget.Clickable // button to activate ground tool
	collisionButton     widget.Clickable // button to activate collision tool
	collisionTestButton widget.Clickable // button to activate collision test tool

	// Collision tool UI
	collisionList        widget.List        // list widget for collision polygons
	collisionButtons     []widget.Clickable // clickable widgets for each polygon in the list
	selectedPolygonIndex int                // index of the currently selected polygon (-1 = none)
	newPolygonButton     widget.Clickable   // button to create a new polygon

	// Collision Test tool state
	collisionTestPoints   []collisionTestResult // history of test results
	collisionTestBSP      *pb.BSPNode           // cached BSP tree
	collisionTestBSPDirty bool                  // true when BSP needs rebuild
}

// NewEditor creates a new level editor instance
func NewEditor(theme *material.Theme, levelFilePath, assetsDir string, level *Level) *Editor {
	// Load the save icon
	saveIcon, err := widget.NewIcon(icons.ContentSave)
	if err != nil {
		log.Printf("Failed to load save icon: %v", err)
	}

	return &Editor{
		theme:           theme,
		levelFilePath:   levelFilePath,
		assetsDir:       assetsDir,
		level:           level,
		assetFiles:      []string{},
		folderStructure: make(map[string][]string),
		folders:         []string{},
		selectedFolder:  "", // root folder
		selectedTexture: "", // no texture selected initially
		toolList: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		assetList: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		folderList: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		folderButtons: []widget.Clickable{},
		assetImages:   make(map[string]image.Image),
		assetButtons:  []widget.Clickable{},
		saveIcon:      saveIcon,
		// Canvas defaults
		gridCellSize: 64.0, // Default cell size in screen pixels
		viewOffsetX:  0.0,
		viewOffsetY:  0.0,
		zoom:         1.0,
		// Tool defaults
		currentTool:             "ground", // Start with ground tool active
		movingPointPolygonIndex: -1,       // No point being moved initially
		movingPointIndex:        -1,       // No point being moved initially
		selectedPolygonIndex:    -1,       // No polygon selected initially
		// Collision list
		collisionList: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		collisionButtons: []widget.Clickable{},
		// Collision Test tool defaults
		collisionTestPoints:   []collisionTestResult{},
		collisionTestBSPDirty: true, // BSP needs to be built initially
	}
}

// GetSelectedTexture returns the relative path of the currently selected texture
// Returns an empty string if no texture is selected
func (e *Editor) GetSelectedTexture() string {
	return e.selectedTexture
}

// HasUnsavedChanges returns true if there are unsaved changes to the level
func (e *Editor) HasUnsavedChanges() bool {
	return e.dirty
}

// Save saves the level to disk and clears the dirty flag
func (e *Editor) Save() error {
	if err := e.level.Save(e.levelFilePath); err != nil {
		return err
	}
	e.dirty = false
	return nil
}

// RequestClose is called when the window close is requested
// Returns true if the window should close, false otherwise
func (e *Editor) RequestClose() bool {
	// If no unsaved changes, allow close immediately
	if !e.dirty {
		return true
	}

	// If we have unsaved changes and haven't shown the dialog yet
	if !e.showCloseDialog {
		e.showCloseDialog = true
		return false // Don't close yet, show dialog first
	}

	// Dialog is showing, return whether user chose to close
	return e.shouldClose
}

// ShouldClose returns true if the window should close
func (e *Editor) ShouldClose() bool {
	return e.shouldClose
}

// currentToolNeedsAssetView returns true if the current tool requires the asset view
func (e *Editor) currentToolNeedsAssetView() bool {
	return e.currentTool == "ground"
}

// currentToolNeedsCollisionList returns true if the current tool requires the collision list view
func (e *Editor) currentToolNeedsCollisionList() bool {
	return e.currentTool == "collision"
}

// buildCollisionBSP builds a BSP tree from the current level's collision polygons
func (e *Editor) buildCollisionBSP() {
	// Convert level.Collisions to bsp.Polygon format
	bspPolygons := make([]bsp.Polygon, 0, len(e.level.Collisions))
	
	for _, collision := range e.level.Collisions {
		// Convert outline points to bsp.Point
		vertices := make([]bsp.Point, 0, len(collision.Outline))
		for _, pt := range collision.Outline {
			vertices = append(vertices, bsp.Point{X: pt.X, Y: pt.Y})
		}
		
		// Only add polygons with at least 3 vertices
		if len(vertices) >= 3 {
			bspPolygons = append(bspPolygons, bsp.Polygon{
				Vertices: vertices,
				IsSolid:  true, // All collision polygons are solid
			})
		}
	}
	
	// Build the BSP tree
	builder := bsp.NewBSPBuilder(bspPolygons)
	e.collisionTestBSP = builder.Build()
	e.collisionTestBSPDirty = false
	
	log.Printf("Built BSP tree from %d collision polygons", len(bspPolygons))
}

// markCollisionBSPDirty marks the BSP tree as needing rebuild
func (e *Editor) markCollisionBSPDirty() {
	e.collisionTestBSPDirty = true
}

// lineTraceBSP performs a line trace through the BSP tree
// Returns true and hit point if the line hits solid geometry
func (e *Editor) lineTraceBSP(fromX, fromY, toX, toY float32) (hit bool, hitX, hitY float32) {
	if e.collisionTestBSP == nil {
		return false, 0, 0
	}
	
	from := bsp.Point{X: fromX, Y: fromY}
	to := bsp.Point{X: toX, Y: toY}
	
	// Recursive line trace through BSP tree
	return e.lineTraceBSPNode(e.collisionTestBSP, from, to, 0.0, 1.0)
}

// lineTraceBSPNode recursively traces a line segment through a BSP node
// t0 and t1 define the parametric range of the line segment [0,1]
func (e *Editor) lineTraceBSPNode(node *pb.BSPNode, from, to bsp.Point, t0, t1 float32) (hit bool, hitX, hitY float32) {
	if node == nil {
		return false, 0, 0
	}
	
	switch n := node.Type.(type) {
	case *pb.BSPNode_Leaf:
		// Leaf node: return hit if solid
		if n.Leaf.IsSolid {
			// Hit! Return the entry point of the line segment
			hitX = from.X + t0*(to.X-from.X)
			hitY = from.Y + t0*(to.Y-from.Y)
			return true, hitX, hitY
		}
		return false, 0, 0
		
	case *pb.BSPNode_Split:
		// Split node: classify line segment endpoints
		split := n.Split
		normalX := split.NormalX
		normalY := split.NormalY
		dist := split.Distance
		
		// Calculate signed distance for both endpoints
		// distance = normal · point - distance
		d0 := normalX*from.X + normalY*from.Y - dist
		d1 := normalX*to.X + normalY*to.Y - dist
		
		epsilon := float32(0.0001)
		
		// Both points on front side
		if d0 > epsilon && d1 > epsilon {
			return e.lineTraceBSPNode(split.Front, from, to, t0, t1)
		}
		
		// Both points on back side
		if d0 <= epsilon && d1 <= epsilon {
			return e.lineTraceBSPNode(split.Back, from, to, t0, t1)
		}
		
		// Line segment spans the plane - need to split it
		// Calculate intersection parameter t where line crosses plane
		// At intersection: d0 + t*(d1-d0) = 0
		// So: t = -d0 / (d1-d0)
		t := -d0 / (d1 - d0)
		tMid := t0 + t*(t1-t0)
		
		// Determine traversal order (near to far)
		var nearNode, farNode *pb.BSPNode
		if d0 > 0 {
			// Start in front
			nearNode = split.Front
			farNode = split.Back
		} else {
			// Start in back
			nearNode = split.Back
			farNode = split.Front
		}
		
		// Check near side first
		hit, hitX, hitY = e.lineTraceBSPNode(nearNode, from, to, t0, tMid)
		if hit {
			return true, hitX, hitY
		}
		
		// Check far side
		return e.lineTraceBSPNode(farNode, from, to, tMid, t1)
	}
	
	return false, 0, 0
}
