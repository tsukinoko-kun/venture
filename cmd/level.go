package cmd

import (
	"image/color"
	"log"
	"os"
	"path/filepath"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/op"
	"gioui.org/widget/material"
	"github.com/bloodmagesoftware/venture/level"
	"github.com/spf13/cobra"
)

var levelCmd = &cobra.Command{
	Use:   "level {level-name}",
	Short: "Edit the specified level",
	Long:  `Creates a new level file if it doesn't exist, then opens the visual editor for that level.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return cmd.Help()
		}
		levelName := args[0]

		projectRoot, err := getProjectRoot()
		if err != nil {
			return err
		}

		levelsDir := filepath.Join(projectRoot, "levels")
		assetsDir := filepath.Join(projectRoot, "assets")
		levelFilePath := filepath.Join(levelsDir, levelName+".yaml")
		lvl := level.New()
		if _, err := os.Stat(levelFilePath); err == nil {
			log.Printf("loading level %s", levelFilePath)
			if err := lvl.Load(levelFilePath); err != nil {
				return err
			}
			log.Printf("loaded level %s", levelFilePath)
		}

		go func() {
			window := new(app.Window)
			window.Perform(system.ActionMaximize)
			err := run(window, levelFilePath, assetsDir, lvl)
			if err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		}()
		app.Main()

		return nil
	},
}

func run(window *app.Window, levelFilePath, assetsDir string, lvl *level.Level) error {
	theme := material.NewTheme()

	// Apply dark mode palette
	theme.Palette = material.Palette{
		Bg:         color.NRGBA{R: 30, G: 30, B: 30, A: 255},    // Dark background
		Fg:         color.NRGBA{R: 220, G: 220, B: 220, A: 255}, // Light text
		ContrastBg: color.NRGBA{R: 50, G: 50, B: 50, A: 255},    // Slightly lighter background
		ContrastFg: color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // White text for contrast
	}

	editor := level.NewEditor(theme, levelFilePath, assetsDir, lvl)

	// Load assets from the assets directory
	if err := editor.LoadAssets(); err != nil {
		log.Printf("warning: failed to load assets: %v", err)
	}

	var ops op.Ops

	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			// Check if we should close
			if editor.RequestClose() {
				return e.Err
			}
			// Don't close yet, continue processing
			window.Invalidate()

		case app.FrameEvent:
			// Check if we should close after dialog interaction
			if editor.ShouldClose() {
				return nil
			}

			// This graphics context is used for managing the rendering state.
			gtx := app.NewContext(&ops, e)

			// Layout the editor
			editor.Layout(gtx)

			// Pass the drawing operations to the GPU.
			e.Frame(gtx.Ops)
		}
	}
}

func init() {
	rootCmd.AddCommand(levelCmd)
}
