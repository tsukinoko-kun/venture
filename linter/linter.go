package linter

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Forbidden imports map - maps forbidden libraries to suggested alternatives
var forbiddenImports = map[string]string{
	"core:thread": "SDL_CreateThread / SDL_Thread",
	"core:sync":   "SDL_CreateMutex / SDL_CreateSemaphore",
	"core:net":    "SDL_Net",
	"core:time":   "SDL_GetTicks / SDL_GetPerformanceCounter",
}

// Regex to find imports
// Matches: import "core:thread"  OR  import th "core:thread"
// Does NOT match comments: // import "core:thread"
var importPattern = regexp.MustCompile(`^\s*import\s+(\w+\s+)?"([^"]+)"`)

// Lint scans Odin source files in srcDir and vendorDir for forbidden imports.
func Lint(srcDir, vendorDir string) error {
	fmt.Println("🔍 Linting imports for console portability...")

	violationCount := 0

	// Scan both source and vendor directories
	for _, scanDir := range []string{srcDir, vendorDir} {
		err := filepath.Walk(scanDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && strings.HasSuffix(path, ".odin") {
				fileErrors, err := checkFileImports(path)
				if err != nil {
					return fmt.Errorf("checking file %s: %w", path, err)
				}

				if len(fileErrors) > 0 {
					for _, errMsg := range fileErrors {
						fmt.Println(errMsg)
						fmt.Println(strings.Repeat("-", 60))
					}
					violationCount += len(fileErrors)
				}
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("walking directory %s: %w", scanDir, err)
		}
	}

	if violationCount > 0 {
		return fmt.Errorf("linter failed: found %d forbidden imports", violationCount)
	}

	fmt.Println("✅ Linter Passed: No forbidden imports found.")
	return nil
}

// checkFileImports checks a single file for forbidden imports.
func checkFileImports(filepath string) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	var errors []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Parse line
		matches := importPattern.FindStringSubmatch(line)
		if matches != nil {
			importedLib := matches[2]

			if alternative, isForbidden := forbiddenImports[importedLib]; isForbidden {
				// Check for explicit bypass comment
				if strings.Contains(line, "// @lint-ignore") {
					continue
				}

				errorMsg := fmt.Sprintf(
					"  [ERROR] File: %s:%d\n"+
						"    Imported: '%s'\n"+
						"    Reason: Forbidden for console portability.\n"+
						"    Solution: Use %s instead.",
					filepath, lineNum, importedLib, alternative,
				)
				errors = append(errors, errorMsg)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning file: %w", err)
	}

	return errors, nil
}

