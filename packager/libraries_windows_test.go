package packager

import (
	"testing"
)

func TestGetSDLDLLs(t *testing.T) {
	tests := []struct {
		name             string
		sdlVersion       string
		sdlTTFVersion    string
		sdlImageVersion  string
		expectedCount    int
		expectedDLLNames []string
	}{
		{
			name:             "All three DLLs",
			sdlVersion:       "3.2.28",
			sdlTTFVersion:    "3.2.2",
			sdlImageVersion:  "3.2.4",
			expectedCount:    3,
			expectedDLLNames: []string{"SDL3.dll", "SDL3_ttf.dll", "SDL3_image.dll"},
		},
		{
			name:             "Only SDL3",
			sdlVersion:       "3.2.28",
			sdlTTFVersion:    "",
			sdlImageVersion:  "",
			expectedCount:    1,
			expectedDLLNames: []string{"SDL3.dll"},
		},
		{
			name:             "No DLLs",
			sdlVersion:       "",
			sdlTTFVersion:    "",
			sdlImageVersion:  "",
			expectedCount:    0,
			expectedDLLNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dlls := GetSDLDLLs(tt.sdlVersion, tt.sdlTTFVersion, tt.sdlImageVersion)

			if len(dlls) != tt.expectedCount {
				t.Errorf("Expected %d DLLs, got %d", tt.expectedCount, len(dlls))
			}

			for i, dll := range dlls {
				if i >= len(tt.expectedDLLNames) {
					t.Errorf("Unexpected DLL: %s", dll.FileName)
					continue
				}
				if dll.FileName != tt.expectedDLLNames[i] {
					t.Errorf("Expected DLL name %s, got %s", tt.expectedDLLNames[i], dll.FileName)
				}
			}
		})
	}
}

func TestSDLVersionURLFormat(t *testing.T) {
	dlls := GetSDLDLLs("3.2.28", "3.2.2", "3.2.4")

	tests := []struct {
		dllIndex        int
		expectedBaseURL string
		expectedArchive string
	}{
		{
			dllIndex:        0,
			expectedBaseURL: "https://github.com/libsdl-org/SDL/releases/download",
			expectedArchive: "SDL3-3.2.28-win32-x64.zip",
		},
		{
			dllIndex:        1,
			expectedBaseURL: "https://github.com/libsdl-org/SDL_ttf/releases/download",
			expectedArchive: "SDL3_ttf-3.2.2-win32-x64.zip",
		},
		{
			dllIndex:        2,
			expectedBaseURL: "https://github.com/libsdl-org/SDL_image/releases/download",
			expectedArchive: "SDL3_image-3.2.4-win32-x64.zip",
		},
	}

	for _, tt := range tests {
		t.Run(dlls[tt.dllIndex].Name, func(t *testing.T) {
			dll := dlls[tt.dllIndex]
			if dll.BaseURL != tt.expectedBaseURL {
				t.Errorf("Expected base URL %s, got %s", tt.expectedBaseURL, dll.BaseURL)
			}
			if dll.ArchiveName != tt.expectedArchive {
				t.Errorf("Expected archive name %s, got %s", tt.expectedArchive, dll.ArchiveName)
			}
		})
	}
}

func TestGetCacheDir(t *testing.T) {
	cacheDir, err := getCacheDir()
	if err != nil {
		t.Fatalf("Failed to get cache directory: %v", err)
	}

	if cacheDir == "" {
		t.Error("Cache directory should not be empty")
	}

	// Should contain "venture" in the path
	if !contains(cacheDir, "venture") {
		t.Errorf("Cache directory should contain 'venture': %s", cacheDir)
	}

	// Should contain "windows-dlls" in the path
	if !contains(cacheDir, "windows-dlls") {
		t.Errorf("Cache directory should contain 'windows-dlls': %s", cacheDir)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
