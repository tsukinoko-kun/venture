package level

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"path/filepath"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// Layout renders the entire editor UI
func (e *Editor) Layout(gtx layout.Context) layout.Dimensions {
	// Register for global keyboard events
	event.Op(gtx.Ops, e)

	// Process keyboard events for deletion mode
	for {
		ev, ok := gtx.Event(key.Filter{Name: "X"})
		if !ok {
			break
		}

		switch ev := ev.(type) {
		case key.Event:
			switch ev.State {
			case key.Press:
				e.isDeleting = true
			case key.Release:
				e.isDeleting = false
			}
		}
	}

	// Process keyboard events for move mode
	for {
		ev, ok := gtx.Event(key.Filter{Name: "M"})
		if !ok {
			break
		}

		switch ev := ev.(type) {
		case key.Event:
			switch ev.State {
			case key.Press:
				e.isMoving = true
			case key.Release:
				e.isMoving = false
				// Stop moving any point when key is released
				e.movingPointPolygonIndex = -1
				e.movingPointIndex = -1
			}
		}
	}

	// Process keyboard events for Escape key (unselect polygon)
	for {
		ev, ok := gtx.Event(key.Filter{Name: key.NameEscape})
		if !ok {
			break
		}

		switch ev := ev.(type) {
		case key.Event:
			if ev.State == key.Press {
				// Unselect the currently selected polygon
				e.selectedPolygonIndex = -1
			}
		}
	}

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
				// Right bar (collision list)
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return e.layoutRightBar(gtx)
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

// layoutTopBar renders the top toolbar with save button and level name
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

// layoutLeftBar renders the left sidebar with tools list
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
					return material.List(e.theme, &e.toolList).Layout(gtx, 3, func(gtx layout.Context, index int) layout.Dimensions {
						return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							var btn *material.ButtonStyle
							var toolName string
							var clickable *widget.Clickable

							if index == 0 {
								toolName = "Ground"
								clickable = &e.groundButton
							} else if index == 1 {
								toolName = "Collision"
								clickable = &e.collisionButton
							} else {
								toolName = "Collision Test"
								clickable = &e.collisionTestButton
							}

							// Handle button clicks
							if clickable.Clicked(gtx) {
								if index == 0 {
									e.currentTool = "ground"
								} else if index == 1 {
									e.currentTool = "collision"
								} else {
									e.currentTool = "collision_test"
									// Clear previous test results when switching to collision test tool
									e.collisionTestPoints = []collisionTestResult{}
								}
							}

							// Create button
							button := material.Button(e.theme, clickable, toolName)

							// Highlight the active tool
							isActive := (index == 0 && e.currentTool == "ground") ||
								(index == 1 && e.currentTool == "collision") ||
								(index == 2 && e.currentTool == "collision_test")
							if isActive {
								button.Background = color.NRGBA{R: 80, G: 140, B: 200, A: 255}
							} else {
								button.Background = color.NRGBA{R: 70, G: 70, B: 70, A: 255}
							}
							button.Color = color.NRGBA{R: 220, G: 220, B: 220, A: 255}

							btn = &button
							return btn.Layout(gtx)
						})
					})
				}),
			)
		},
	)
}

// layoutBottomBar renders the bottom panel with folder and asset lists
func (e *Editor) layoutBottomBar(gtx layout.Context) layout.Dimensions {
	// Only show the bottom bar if the current tool needs the asset view
	if !e.currentToolNeedsAssetView() {
		// Return empty dimensions when asset view is not needed
		return layout.Dimensions{}
	}

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

// layoutRightBar renders the right sidebar with collision polygon list (only visible in collision tool mode)
func (e *Editor) layoutRightBar(gtx layout.Context) layout.Dimensions {
	// Only show the right bar if the current tool needs the collision list
	if !e.currentToolNeedsCollisionList() {
		// Return empty dimensions when collision list is not needed
		return layout.Dimensions{}
	}

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
						label := material.H6(e.theme, "Collision Polygons")
						label.Color = color.NRGBA{R: 220, G: 220, B: 220, A: 255}
						return label.Layout(gtx)
					})
				}),
				// Collision polygon list
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					numPolygons := len(e.level.Collisions)

					// Ensure we have enough clickable widgets for all polygons
					for len(e.collisionButtons) < numPolygons {
						e.collisionButtons = append(e.collisionButtons, widget.Clickable{})
					}

					return material.List(e.theme, &e.collisionList).Layout(gtx, numPolygons, func(gtx layout.Context, index int) layout.Dimensions {
						return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							// Handle button clicks
							if e.collisionButtons[index].Clicked(gtx) {
								e.selectedPolygonIndex = index
							}

							polygonName := fmt.Sprintf("Polygon %d", index+1)

							// Create button
							button := material.Button(e.theme, &e.collisionButtons[index], polygonName)

							// Highlight the selected polygon
							if index == e.selectedPolygonIndex {
								button.Background = color.NRGBA{R: 80, G: 140, B: 200, A: 255}
							} else {
								button.Background = color.NRGBA{R: 70, G: 70, B: 70, A: 255}
							}
							button.Color = color.NRGBA{R: 220, G: 220, B: 220, A: 255}

							return button.Layout(gtx)
						})
					})
				}),
				// New Polygon button
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						// Handle button click
						if e.newPolygonButton.Clicked(gtx) {
							// Create a new empty polygon
							newPolygon := Polygon{
								Outline: make([]Vec2, 0),
							}
							e.level.Collisions = append(e.level.Collisions, newPolygon)
							// Select the newly created polygon
							e.selectedPolygonIndex = len(e.level.Collisions) - 1
							// Mark as dirty
							e.dirty = true
							e.markCollisionBSPDirty() // Mark BSP as needing rebuild
							log.Printf("Created new polygon at index %d", e.selectedPolygonIndex)
						}

						// Create button with blue background
						button := material.Button(e.theme, &e.newPolygonButton, "+ New Polygon")
						button.Background = color.NRGBA{R: 60, G: 120, B: 200, A: 255}
						button.Color = color.NRGBA{R: 220, G: 220, B: 220, A: 255}

						return button.Layout(gtx)
					})
				}),
			)
		},
	)
}
