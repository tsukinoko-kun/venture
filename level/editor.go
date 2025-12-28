package level

import (
	"image"
	"log"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

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
	currentTool     string           // currently active tool ("ground" or "collision")
	groundButton    widget.Clickable // button to activate ground tool
	collisionButton widget.Clickable // button to activate collision tool

	// Collision tool UI
	collisionList        widget.List        // list widget for collision polygons
	collisionButtons     []widget.Clickable // clickable widgets for each polygon in the list
	selectedPolygonIndex int                // index of the currently selected polygon (-1 = none)
	newPolygonButton     widget.Clickable   // button to create a new polygon
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
