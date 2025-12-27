package level

import (
	"image/color"
	"io/fs"
	"path/filepath"
	"strings"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type Editor struct {
	theme         *material.Theme
	levelFilePath string
	assetsDir     string
	level         *Level
	
	// UI state
	assetFiles []string
	toolList   widget.List
	assetList  widget.List
}

func NewEditor(theme *material.Theme, levelFilePath, assetsDir string, level *Level) *Editor {
	return &Editor{
		theme:         theme,
		levelFilePath: levelFilePath,
		assetsDir:     assetsDir,
		level:         level,
		assetFiles:    []string{},
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
	}
}

func (e *Editor) LoadAssets() error {
	e.assetFiles = []string{}
	
	err := filepath.WalkDir(e.assetsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			relPath, err := filepath.Rel(e.assetsDir, path)
			if err != nil {
				return err
			}
			e.assetFiles = append(e.assetFiles, relPath)
		}
		return nil
	})
	
	return err
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
				Axis: layout.Vertical,
			}.Layout(gtx,
				// Header
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						label := material.H6(e.theme, "Assets")
						label.Color = color.NRGBA{R: 220, G: 220, B: 220, A: 255}
						return label.Layout(gtx)
					})
				}),
				// Asset file list
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return material.List(e.theme, &e.assetList).Layout(gtx, len(e.assetFiles), func(gtx layout.Context, index int) layout.Dimensions {
						return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							// Indent based on directory depth
							relPath := e.assetFiles[index]
							depth := strings.Count(relPath, string(filepath.Separator))
							indent := unit.Dp(float32(depth) * 16)
							
							return layout.Inset{Left: indent + unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								label := material.Body2(e.theme, filepath.Base(relPath))
								label.Color = color.NRGBA{R: 180, G: 180, B: 180, A: 255}
								return label.Layout(gtx)
							})
						})
					})
				}),
			)
		},
	)
}

