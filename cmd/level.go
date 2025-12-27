package cmd

import (
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
	editor := level.NewEditor(theme, levelFilePath, assetsDir, lvl)
	
	// Load assets from the assets directory
	if err := editor.LoadAssets(); err != nil {
		log.Printf("warning: failed to load assets: %v", err)
	}
	
	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
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
