package level

import (
	"image"
	"image/color"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/xfmoulet/qoi"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

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
}

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
	}
}

func (e *Editor) LoadAssets() error {
	e.assetFiles = []string{}
	e.folderStructure = make(map[string][]string)
	e.folders = []string{}
	e.folderButtons = []widget.Clickable{}
	e.assetImages = make(map[string]image.Image)
	e.assetButtons = []widget.Clickable{}

	// Track unique folders
	folderSet := make(map[string]bool)

	err := filepath.WalkDir(e.assetsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				return fs.SkipDir
			}
			// Add folder to the set (but continue walking into it)
			relPath, err := filepath.Rel(e.assetsDir, path)
			if err != nil {
				return err
			}
			if relPath != "." {
				folderSet[relPath] = true
			}
			return nil // Continue walking into this directory
		}

		name := d.Name()
		// Skip non-asset files
		if isIgnoredFile(name) {
			return nil
		}

		relPath, err := filepath.Rel(e.assetsDir, path)
		if err != nil {
			return err
		}
		e.assetFiles = append(e.assetFiles, relPath)

		// Get the folder this file belongs to
		folderPath := filepath.Dir(relPath)
		if folderPath == "." {
			folderPath = "" // root folder
		}

		e.folderStructure[folderPath] = append(e.folderStructure[folderPath], relPath)

		return nil
	})

	if err != nil {
		return err
	}

	// Convert folder set to sorted list
	e.folders = make([]string, 0, len(folderSet)+1)
	e.folders = append(e.folders, "") // Add root folder first
	for folder := range folderSet {
		e.folders = append(e.folders, folder)
	}

	// Create clickable widgets for each folder
	e.folderButtons = make([]widget.Clickable, len(e.folders))

	return nil
}

// loadAssetImage loads and decodes a QOI image file, caching the result
func (e *Editor) loadAssetImage(relPath string) (image.Image, error) {
	// Check cache first
	if img, ok := e.assetImages[relPath]; ok {
		return img, nil
	}

	// Load from disk
	fullPath := filepath.Join(e.assetsDir, relPath)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode QOI image
	img, err := qoi.Decode(file)
	if err != nil {
		return nil, err
	}

	// Cache the image
	e.assetImages[relPath] = img

	return img, nil
}

func (e *Editor) layoutFolderList(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
	gtx.Constraints.Max.X = gtx.Constraints.Min.X

	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: color.NRGBA{R: 50, G: 50, B: 50, A: 255}}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(gtx,
				// Header
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						label := material.H6(e.theme, "Folders")
						label.Color = color.NRGBA{R: 220, G: 220, B: 220, A: 255}
						return label.Layout(gtx)
					})
				}),
				// Folder list
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return material.List(e.theme, &e.folderList).Layout(gtx, len(e.folders), func(gtx layout.Context, index int) layout.Dimensions {
						folder := e.folders[index]

						// Handle button clicks
						if e.folderButtons[index].Clicked(gtx) {
							e.selectedFolder = folder
						}

						// Determine display name
						displayName := folder
						if displayName == "" {
							displayName = "(root)"
						}

						// Check if this folder is selected
						isSelected := e.selectedFolder == folder

						return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return material.Clickable(gtx, &e.folderButtons[index], func(gtx layout.Context) layout.Dimensions {
								// Paint selection background if selected
								return layout.Stack{}.Layout(gtx,
									layout.Expanded(func(gtx layout.Context) layout.Dimensions {
										if isSelected {
											defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
											paint.ColorOp{Color: color.NRGBA{R: 70, G: 120, B: 180, A: 255}}.Add(gtx.Ops)
											paint.PaintOp{}.Add(gtx.Ops)
										}
										return layout.Dimensions{Size: gtx.Constraints.Min}
									}),
									layout.Stacked(func(gtx layout.Context) layout.Dimensions {
										return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											label := material.Body2(e.theme, displayName)
											if isSelected {
												label.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
											} else {
												label.Color = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
											}
											return label.Layout(gtx)
										})
									}),
								)
							})
						})
					})
				}),
			)
		},
	)
}

// layoutAssetTile renders a single asset tile with thumbnail and filename
func (e *Editor) layoutAssetTile(gtx layout.Context, index int, relPath, fileName string) layout.Dimensions {
	// Load the image (from cache or disk)
	img, err := e.loadAssetImage(relPath)

	// Handle click events to select texture
	if e.assetButtons[index].Clicked(gtx) {
		e.selectedTexture = relPath
	}

	// Check if this texture is currently selected
	isSelected := e.selectedTexture == relPath

	return material.Clickable(gtx, &e.assetButtons[index], func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(gtx,
				// Thumbnail preview
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// Fixed size square for thumbnail
					size := gtx.Dp(unit.Dp(96))

					// Set constraints to fixed square
					gtx.Constraints.Min.X = size
					gtx.Constraints.Max.X = size
					gtx.Constraints.Min.Y = size
					gtx.Constraints.Max.Y = size

					// Create a stack for layering selection highlight and image
					return layout.Stack{}.Layout(gtx,
						// Background layer
						layout.Expanded(func(gtx layout.Context) layout.Dimensions {
							defer clip.Rect{Max: image.Point{X: size, Y: size}}.Push(gtx.Ops).Pop()
							// Use different background color when selected
							if isSelected {
								paint.ColorOp{Color: color.NRGBA{R: 70, G: 120, B: 180, A: 255}}.Add(gtx.Ops)
							} else {
								paint.ColorOp{Color: color.NRGBA{R: 60, G: 60, B: 60, A: 255}}.Add(gtx.Ops)
							}
							paint.PaintOp{}.Add(gtx.Ops)
							return layout.Dimensions{Size: image.Point{X: size, Y: size}}
						}),
						// Image layer
						layout.Stacked(func(gtx layout.Context) layout.Dimensions {
							if err != nil {
								// Show error placeholder
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									label := material.Body2(e.theme, "?")
									label.Color = color.NRGBA{R: 150, G: 150, B: 150, A: 255}
									return label.Layout(gtx)
								})
							}

							// Render the image centered with preserved aspect ratio
							imgSize := img.Bounds().Size()
							maxDim := max(imgSize.X, imgSize.Y)
							if maxDim == 0 {
								maxDim = 1
							}
							scale := float32(size) / float32(maxDim)

							// Use widget.Image to properly render with scaling
							return widget.Image{
								Src:      paint.NewImageOp(img),
								Fit:      widget.Contain, // Preserve aspect ratio and fit within bounds
								Scale:    scale,
								Position: layout.Center,
							}.Layout(gtx)
						}),
						// Selection border overlay
						layout.Expanded(func(gtx layout.Context) layout.Dimensions {
							if isSelected {
								// Draw a border around the selected texture
								borderWidth := 3
								rect := image.Rectangle{Max: image.Point{X: size, Y: size}}

								// Draw top border
								topRect := image.Rectangle{
									Min: rect.Min,
									Max: image.Point{X: rect.Max.X, Y: rect.Min.Y + borderWidth},
								}
								defer clip.Rect(topRect).Push(gtx.Ops).Pop()
								paint.ColorOp{Color: color.NRGBA{R: 100, G: 180, B: 255, A: 255}}.Add(gtx.Ops)
								paint.PaintOp{}.Add(gtx.Ops)

								// Draw bottom border
								bottomRect := image.Rectangle{
									Min: image.Point{X: rect.Min.X, Y: rect.Max.Y - borderWidth},
									Max: rect.Max,
								}
								defer clip.Rect(bottomRect).Push(gtx.Ops).Pop()
								paint.ColorOp{Color: color.NRGBA{R: 100, G: 180, B: 255, A: 255}}.Add(gtx.Ops)
								paint.PaintOp{}.Add(gtx.Ops)

								// Draw left border
								leftRect := image.Rectangle{
									Min: rect.Min,
									Max: image.Point{X: rect.Min.X + borderWidth, Y: rect.Max.Y},
								}
								defer clip.Rect(leftRect).Push(gtx.Ops).Pop()
								paint.ColorOp{Color: color.NRGBA{R: 100, G: 180, B: 255, A: 255}}.Add(gtx.Ops)
								paint.PaintOp{}.Add(gtx.Ops)

								// Draw right border
								rightRect := image.Rectangle{
									Min: image.Point{X: rect.Max.X - borderWidth, Y: rect.Min.Y},
									Max: rect.Max,
								}
								defer clip.Rect(rightRect).Push(gtx.Ops).Pop()
								paint.ColorOp{Color: color.NRGBA{R: 100, G: 180, B: 255, A: 255}}.Add(gtx.Ops)
								paint.PaintOp{}.Add(gtx.Ops)
							}
							return layout.Dimensions{Size: image.Point{X: size, Y: size}}
						}),
					)
				}),
				// Filename label
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						// Limit width to tile size
						gtx.Constraints.Max.X = gtx.Dp(unit.Dp(96))
						label := material.Caption(e.theme, fileName)
						// Highlight selected texture's filename
						if isSelected {
							label.Color = color.NRGBA{R: 100, G: 180, B: 255, A: 255}
						} else {
							label.Color = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
						}
						label.Alignment = 1 // center aligned (text.Middle)
						return label.Layout(gtx)
					})
				}),
			)
		})
	})
}

func (e *Editor) layoutAssetList(gtx layout.Context) layout.Dimensions {
	// Get files in the selected folder
	filesInFolder := e.folderStructure[e.selectedFolder]

	// Ensure we have enough buttons for all assets in this folder
	if len(e.assetButtons) < len(filesInFolder) {
		e.assetButtons = make([]widget.Clickable, len(filesInFolder))
	}

	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: color.NRGBA{R: 45, G: 45, B: 45, A: 255}}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(gtx,
				// Header
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						headerText := "Assets"
						if e.selectedFolder != "" {
							headerText = "Assets in: " + e.selectedFolder
						} else {
							headerText = "Assets in: (root)"
						}
						label := material.H6(e.theme, headerText)
						label.Color = color.NRGBA{R: 220, G: 220, B: 220, A: 255}
						return label.Layout(gtx)
					})
				}),
				// Asset grid
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					if len(filesInFolder) == 0 {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							label := material.Body2(e.theme, "No assets in this folder")
							label.Color = color.NRGBA{R: 150, G: 150, B: 150, A: 255}
							return label.Layout(gtx)
						})
					}

					// Calculate grid layout
					tileWidth := gtx.Dp(unit.Dp(104)) // 96 + 8 padding
					padding := gtx.Dp(unit.Dp(8))
					availableWidth := gtx.Constraints.Max.X - padding*2
					columns := max(1, availableWidth/tileWidth)

					// Use a scrollable list for rows
					return layout.Inset{
						Top:    unit.Dp(8),
						Bottom: unit.Dp(8),
						Left:   unit.Dp(8),
						Right:  unit.Dp(8),
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						// Calculate number of rows
						rows := (len(filesInFolder) + columns - 1) / columns

						return material.List(e.theme, &e.assetList).Layout(gtx, rows, func(gtx layout.Context, row int) layout.Dimensions {
							// Layout one row of tiles
							return layout.Flex{
								Axis: layout.Horizontal,
							}.Layout(gtx,
								// Generate flex children for each column in this row
								func() []layout.FlexChild {
									children := make([]layout.FlexChild, 0, columns)
									for col := 0; col < columns; col++ {
										index := row*columns + col
										if index >= len(filesInFolder) {
											break
										}

										// Capture variables for closure
										captureIndex := index
										captureRelPath := filesInFolder[index]
										captureFileName := filepath.Base(filesInFolder[index])

										children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return e.layoutAssetTile(gtx, captureIndex, captureRelPath, captureFileName)
										}))
									}
									return children
								}()...,
							)
						})
					})
				}),
			)
		},
	)
}

// isIgnoredFile returns true for files that should not be treated as assets.
func isIgnoredFile(name string) bool {
	// Hidden files (starting with .)
	if strings.HasPrefix(name, ".") {
		return true
	}
	// Windows thumbnail cache
	if strings.EqualFold(name, "Thumbs.db") {
		return true
	}
	// Windows desktop.ini
	if strings.EqualFold(name, "desktop.ini") {
		return true
	}
	return false
}

func (e *Editor) Layout(gtx layout.Context) layout.Dimensions {
	// Handle close dialog buttons
	if e.closeSaveButton.Clicked(gtx) {
		if err := e.Save(); err != nil {
			log.Printf("Failed to save level: %v", err)
		} else {
			log.Printf("Level saved to %s", e.levelFilePath)
		}
		e.showCloseDialog = false
		e.shouldClose = true
	}
	if e.closeDiscardButton.Clicked(gtx) {
		e.showCloseDialog = false
		e.shouldClose = true
	}

	dims := layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		// Top bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return e.layoutTopBar(gtx)
		}),
		// Middle section (left bar + canvas)
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis: layout.Horizontal,
			}.Layout(gtx,
				// Left bar
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return e.layoutLeftBar(gtx)
				}),
				// Canvas
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return e.layoutCanvas(gtx)
				}),
			)
		}),
		// Bottom bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return e.layoutBottomBar(gtx)
		}),
	)

	// Draw close confirmation dialog on top if needed
	if e.showCloseDialog {
		e.layoutCloseDialog(gtx)
	}

	return dims
}

func (e *Editor) layoutTopBar(gtx layout.Context) layout.Dimensions {
	// Background
	gtx.Constraints.Min = gtx.Constraints.Max
	gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(40))
	gtx.Constraints.Max.Y = gtx.Constraints.Min.Y

	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: color.NRGBA{R: 40, G: 40, B: 40, A: 255}}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
			}.Layout(gtx,
				// Level name
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						filename := filepath.Base(e.levelFilePath)
						title := "Level: " + filename
						if e.dirty {
							title += " *" // Add asterisk for unsaved changes
						}
						label := material.Body1(e.theme, title)
						label.Color = color.NRGBA{R: 220, G: 220, B: 220, A: 255}
						return label.Layout(gtx)
					})
				}),
				// Spacer
				layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
				// Save button
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// Handle save button clicks
					if e.saveButton.Clicked(gtx) {
						if err := e.Save(); err != nil {
							log.Printf("Failed to save level: %v", err)
						} else {
							log.Printf("Level saved to %s", e.levelFilePath)
						}
					}

					// Only show button if icon loaded successfully
					if e.saveIcon != nil {
						btn := material.IconButton(e.theme, &e.saveButton, e.saveIcon, "Save level")
						// Change button color based on dirty state
						if e.dirty {
							btn.Background = color.NRGBA{R: 200, G: 120, B: 60, A: 255} // Orange when dirty
						} else {
							btn.Background = color.NRGBA{R: 60, G: 120, B: 200, A: 255} // Blue when clean
						}
						btn.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
						btn.Size = unit.Dp(20)
						return btn.Layout(gtx)
					}
					return layout.Dimensions{}
				}),
			)
		},
	)
}

func (e *Editor) layoutLeftBar(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
	gtx.Constraints.Max.X = gtx.Constraints.Min.X

	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: color.NRGBA{R: 50, G: 50, B: 50, A: 255}}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(gtx,
				// Header
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						label := material.H6(e.theme, "Tools")
						label.Color = color.NRGBA{R: 220, G: 220, B: 220, A: 255}
						return label.Layout(gtx)
					})
				}),
				// Tool list
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return material.List(e.theme, &e.toolList).Layout(gtx, 1, func(gtx layout.Context, index int) layout.Dimensions {
						return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							label := material.Body1(e.theme, "Ground")
							label.Color = color.NRGBA{R: 180, G: 180, B: 180, A: 255}
							return label.Layout(gtx)
						})
					})
				}),
			)
		},
	)
}

func (e *Editor) layoutCanvas(gtx layout.Context) layout.Dimensions {
	// Handle pointer input for panning and zooming
	e.handleCanvasInput(gtx)

	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: color.NRGBA{R: 60, G: 60, B: 60, A: 255}}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
		func(gtx layout.Context) layout.Dimensions {
			// Draw the ground editor grid
			e.drawGroundGrid(gtx)
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
			switch ev.Kind {
			case pointer.Press:
				// Start panning on right mouse button press
				if ev.Buttons == pointer.ButtonSecondary {
					e.isPanning = true
					e.lastMouseX = ev.Position.X
					e.lastMouseY = ev.Position.Y
				}
				// Place texture on left mouse button press
				if ev.Buttons == pointer.ButtonPrimary {
					e.placeTileAtPosition(gtx, ev.Position.X, ev.Position.Y)
				}

			case pointer.Release:
				// Stop panning on right mouse button release
				if ev.Buttons&pointer.ButtonSecondary == 0 {
					e.isPanning = false
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
				// Place texture while dragging with left mouse button
				if ev.Buttons == pointer.ButtonPrimary {
					e.placeTileAtPosition(gtx, ev.Position.X, ev.Position.Y)
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

// drawGroundGrid draws the grid and origin cross for the ground editor
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

	// Calculate the range of grid cells to draw (visible area)
	// We need to figure out which grid cells are visible on screen
	minGridX := int32(math.Floor(float64((-centerX - e.viewOffsetX) / cellSize)))
	maxGridX := int32(math.Ceil(float64((canvasWidth - centerX - e.viewOffsetX) / cellSize)))
	minGridY := int32(math.Floor(float64((-centerY - e.viewOffsetY) / cellSize)))
	maxGridY := int32(math.Ceil(float64((canvasHeight - centerY - e.viewOffsetY) / cellSize)))

	// Limit the grid range to prevent drawing too many lines
	const maxGridRange = 100
	if minGridX < -maxGridRange {
		minGridX = -maxGridRange
	}
	if maxGridX > maxGridRange {
		maxGridX = maxGridRange
	}
	if minGridY < -maxGridRange {
		minGridY = -maxGridRange
	}
	if maxGridY > maxGridRange {
		maxGridY = maxGridRange
	}

	// Draw placed tiles first (underneath the grid)
	e.drawPlacedTiles(gtx, centerX, centerY, cellSize)

	// Draw vertical grid lines
	gridColor := color.NRGBA{R: 80, G: 80, B: 80, A: 255}
	for x := minGridX; x <= maxGridX; x++ {
		screenX := centerX + e.viewOffsetX + float32(x)*cellSize

		// Only draw if the line is within canvas bounds
		if screenX >= 0 && screenX <= canvasWidth {
			p1 := f32.Point{X: screenX, Y: 0}
			p2 := f32.Point{X: screenX, Y: canvasHeight}
			e.drawLine(gtx, p1, p2, 1, gridColor)
		}
	}

	// Draw horizontal grid lines
	for y := minGridY; y <= maxGridY; y++ {
		screenY := centerY + e.viewOffsetY + float32(y)*cellSize

		// Only draw if the line is within canvas bounds
		if screenY >= 0 && screenY <= canvasHeight {
			p1 := f32.Point{X: 0, Y: screenY}
			p2 := f32.Point{X: canvasWidth, Y: screenY}
			e.drawLine(gtx, p1, p2, 1, gridColor)
		}
	}

	// Draw origin cross (thicker and different color)
	originColor := color.NRGBA{R: 255, G: 100, B: 100, A: 255}
	originX := centerX + e.viewOffsetX
	originY := centerY + e.viewOffsetY

	// Only draw the cross if it's within or near the visible canvas
	if originX >= -20 && originX <= canvasWidth+20 && originY >= -20 && originY <= canvasHeight+20 {
		// Vertical line of cross
		crossSize := float32(20.0)
		p1 := f32.Point{X: originX, Y: originY - crossSize}
		p2 := f32.Point{X: originX, Y: originY + crossSize}
		e.drawLine(gtx, p1, p2, 3, originColor)

		// Horizontal line of cross
		p1 = f32.Point{X: originX - crossSize, Y: originY}
		p2 = f32.Point{X: originX + crossSize, Y: originY}
		e.drawLine(gtx, p1, p2, 3, originColor)
	}
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

// drawLine draws a line between two points with a given width and color
func (e *Editor) drawLine(gtx layout.Context, p1, p2 f32.Point, width float32, col color.NRGBA) {
	// Calculate the direction vector
	dx := p2.X - p1.X
	dy := p2.Y - p1.Y
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))

	if length == 0 {
		return // Can't draw a zero-length line
	}

	// Normalize the direction
	dx /= length
	dy /= length

	// Perpendicular vector for width
	perpX := -dy * width / 2.0
	perpY := dx * width / 2.0

	// Create a quad (rectangle) for the line
	var path clip.Path
	path.Begin(gtx.Ops)

	// Move to first corner
	path.MoveTo(f32.Point{X: p1.X + perpX, Y: p1.Y + perpY})
	// Line to second corner
	path.LineTo(f32.Point{X: p2.X + perpX, Y: p2.Y + perpY})
	// Line to third corner
	path.LineTo(f32.Point{X: p2.X - perpX, Y: p2.Y - perpY})
	// Line to fourth corner
	path.LineTo(f32.Point{X: p1.X - perpX, Y: p1.Y - perpY})
	// Close the path
	path.Close()

	// Draw the filled path
	defer clip.Outline{Path: path.End()}.Op().Push(gtx.Ops).Pop()
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

func (e *Editor) layoutBottomBar(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max
	gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(200))
	gtx.Constraints.Max.Y = gtx.Constraints.Min.Y

	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: color.NRGBA{R: 45, G: 45, B: 45, A: 255}}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis: layout.Horizontal,
			}.Layout(gtx,
				// Left side: folder structure
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return e.layoutFolderList(gtx)
				}),
				// Right side: asset files in selected folder
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return e.layoutAssetList(gtx)
				}),
			)
		},
	)
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

// layoutCloseDialog renders the close confirmation dialog
func (e *Editor) layoutCloseDialog(gtx layout.Context) layout.Dimensions {
	// Semi-transparent overlay
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: color.NRGBA{R: 0, G: 0, B: 0, A: 200}}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	// Center the dialog
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Dialog box
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				// Dark dialog background
				defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Max}, 8).Push(gtx.Ops).Pop()
				paint.ColorOp{Color: color.NRGBA{R: 45, G: 45, B: 45, A: 255}}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				return layout.Dimensions{Size: gtx.Constraints.Max}
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(24)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis: layout.Vertical,
					}.Layout(gtx,
						// Title
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Bottom: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								label := material.H6(e.theme, "Unsaved Changes")
								label.Color = color.NRGBA{R: 240, G: 240, B: 240, A: 255}
								return label.Layout(gtx)
							})
						}),
						// Message
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Bottom: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								gtx.Constraints.Max.X = gtx.Dp(unit.Dp(400))
								label := material.Body1(e.theme, "Do you want to save your changes before closing?")
								label.Color = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
								return label.Layout(gtx)
							})
						}),
						// Buttons
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{
								Axis:    layout.Horizontal,
								Spacing: layout.SpaceEnd,
							}.Layout(gtx,
								// Save button
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									btn := material.Button(e.theme, &e.closeSaveButton, "Save")
									btn.Background = color.NRGBA{R: 60, G: 120, B: 200, A: 255}
									btn.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
									return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, btn.Layout)
								}),
								// Discard button
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									btn := material.Button(e.theme, &e.closeDiscardButton, "Discard")
									btn.Background = color.NRGBA{R: 200, G: 80, B: 60, A: 255}
									btn.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
									return btn.Layout(gtx)
								}),
							)
						}),
					)
				})
			},
		)
	})
}
