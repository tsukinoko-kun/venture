package level

import (
	"image"
	"image/color"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/xfmoulet/qoi"
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
	toolList        widget.List
	assetList       widget.List
	folderList      widget.List
	folderButtons   []widget.Clickable     // clickable widgets for each folder
	assetImages     map[string]image.Image // cache of loaded QOI textures
	assetButtons    []widget.Clickable     // clickable widgets for each asset in grid
}

func NewEditor(theme *material.Theme, levelFilePath, assetsDir string, level *Level) *Editor {
	return &Editor{
		theme:           theme,
		levelFilePath:   levelFilePath,
		assetsDir:       assetsDir,
		level:           level,
		assetFiles:      []string{},
		folderStructure: make(map[string][]string),
		folders:         []string{},
		selectedFolder:  "", // root folder
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

	// Handle click events
	clicked := e.assetButtons[index].Clicked(gtx)
	_ = clicked // TODO: handle selection

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

					// Draw background
					defer clip.Rect{Max: image.Point{X: size, Y: size}}.Push(gtx.Ops).Pop()
					paint.ColorOp{Color: color.NRGBA{R: 60, G: 60, B: 60, A: 255}}.Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)

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
				// Filename label
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						// Limit width to tile size
						gtx.Constraints.Max.X = gtx.Dp(unit.Dp(96))
						label := material.Caption(e.theme, fileName)
						label.Color = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
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
	return layout.Flex{
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
			return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				filename := filepath.Base(e.levelFilePath)
				label := material.Body1(e.theme, "Level: "+filename)
				label.Color = color.NRGBA{R: 220, G: 220, B: 220, A: 255}
				return label.Layout(gtx)
			})
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
	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: color.NRGBA{R: 60, G: 60, B: 60, A: 255}}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := material.Body1(e.theme, "Canvas Area")
				label.Color = color.NRGBA{R: 150, G: 150, B: 150, A: 255}
				return label.Layout(gtx)
			})
		},
	)
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
