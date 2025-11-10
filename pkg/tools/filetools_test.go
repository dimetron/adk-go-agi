package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileReadTool(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		relativePath string
		setupFunc    func(t *testing.T, workspaceDir string)
		wantErr      bool
		errContains  string
	}{
		{
			name:         "read existing file",
			content:      "Hello, World!",
			relativePath: "test.txt",
			setupFunc: func(t *testing.T, workspaceDir string) {
				t.Helper()
				filePath := filepath.Join(workspaceDir, "test.txt")
				if err := os.WriteFile(filePath, []byte("Hello, World!"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name:         "read non-existent file",
			content:      "",
			relativePath: "non-existent.txt",
			setupFunc:    func(t *testing.T, workspaceDir string) {},
			wantErr:      true,
			errContains:  "failed to read file",
		},
		{
			name:         "read file with unicode content",
			content:      "Hello, 世界! 🌍",
			relativePath: "unicode.txt",
			setupFunc: func(t *testing.T, workspaceDir string) {
				t.Helper()
				filePath := filepath.Join(workspaceDir, "unicode.txt")
				if err := os.WriteFile(filePath, []byte("Hello, 世界! 🌍"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name:         "read empty file",
			content:      "",
			relativePath: "empty.txt",
			setupFunc: func(t *testing.T, workspaceDir string) {
				t.Helper()
				filePath := filepath.Join(workspaceDir, "empty.txt")
				if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name:         "read file in subdirectory",
			content:      "Nested content",
			relativePath: "subdir/nested.txt",
			setupFunc: func(t *testing.T, workspaceDir string) {
				t.Helper()
				subdir := filepath.Join(workspaceDir, "subdir")
				if err := os.MkdirAll(subdir, 0755); err != nil {
					t.Fatalf("failed to create subdirectory: %v", err)
				}
				filePath := filepath.Join(subdir, "nested.txt")
				if err := os.WriteFile(filePath, []byte("Nested content"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name:         "prevent path traversal with ..",
			content:      "",
			relativePath: "../outside.txt",
			setupFunc:    func(t *testing.T, workspaceDir string) {},
			wantErr:      true,
			errContains:  "path traversal detected",
		},
		{
			name:         "prevent absolute path",
			content:      "",
			relativePath: "/etc/passwd",
			setupFunc:    func(t *testing.T, workspaceDir string) {},
			wantErr:      true,
			errContains:  "absolute paths are not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary workspace directory
			workspaceDir, err := os.MkdirTemp("", "filetools-workspace-*")
			if err != nil {
				t.Fatalf("failed to create workspace dir: %v", err)
			}
			defer func(path string) {
				_ = os.RemoveAll(path)
			}(workspaceDir)

			// Setup test file
			tt.setupFunc(t, workspaceDir)

			// Execute the file read directly
			input := FileReadInput{Path: tt.relativePath}
			output, err := executeFileRead(workspaceDir, input)

			// Check error expectations
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Execute() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			// Check output
			if output.Content != tt.content {
				t.Errorf("Execute() content = %q, want %q", output.Content, tt.content)
			}

			if output.Path != tt.relativePath {
				t.Errorf("Execute() path = %q, want %q", output.Path, tt.relativePath)
			}
		})
	}
}

func TestFileWriteTool(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		relativePath string
		setupFunc    func(t *testing.T, workspaceDir string)
		wantErr      bool
		errContains  string
	}{
		{
			name:         "write to new file",
			content:      "Hello, World!",
			relativePath: "test.txt",
			setupFunc:    func(t *testing.T, workspaceDir string) {},
			wantErr:      false,
		},
		{
			name:         "write unicode content",
			content:      "Hello, 世界! 🌍",
			relativePath: "unicode.txt",
			setupFunc:    func(t *testing.T, workspaceDir string) {},
			wantErr:      false,
		},
		{
			name:         "write empty content",
			content:      "",
			relativePath: "empty.txt",
			setupFunc:    func(t *testing.T, workspaceDir string) {},
			wantErr:      false,
		},
		{
			name:         "write to nested directory",
			content:      "Nested content",
			relativePath: "subdir/nested/test.txt",
			setupFunc:    func(t *testing.T, workspaceDir string) {},
			wantErr:      false,
		},
		{
			name:         "overwrite existing file",
			content:      "New content",
			relativePath: "overwrite.txt",
			setupFunc: func(t *testing.T, workspaceDir string) {
				t.Helper()
				filePath := filepath.Join(workspaceDir, "overwrite.txt")
				if err := os.WriteFile(filePath, []byte("Old content"), 0644); err != nil {
					t.Fatalf("failed to create existing file: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name:         "prevent path traversal with ..",
			content:      "Malicious content",
			relativePath: "../outside.txt",
			setupFunc:    func(t *testing.T, workspaceDir string) {},
			wantErr:      true,
			errContains:  "path traversal detected",
		},
		{
			name:         "prevent absolute path",
			content:      "Malicious content",
			relativePath: "/tmp/outside.txt",
			setupFunc:    func(t *testing.T, workspaceDir string) {},
			wantErr:      true,
			errContains:  "absolute paths are not allowed",
		},
		{
			name:         "prevent complex path traversal",
			content:      "Malicious content",
			relativePath: "subdir/../../outside.txt",
			setupFunc:    func(t *testing.T, workspaceDir string) {},
			wantErr:      true,
			errContains:  "path traversal detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary workspace directory
			workspaceDir, err := os.MkdirTemp("", "filetools-workspace-*")
			if err != nil {
				t.Fatalf("failed to create workspace dir: %v", err)
			}
			defer func(path string) {
				_ = os.RemoveAll(path)
			}(workspaceDir)

			// Setup test
			tt.setupFunc(t, workspaceDir)

			// Execute the file write directly
			input := FileWriteInput{
				Path:    tt.relativePath,
				Content: tt.content,
			}
			output, err := executeFileWrite(workspaceDir, input)

			// Check error expectations
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Execute() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			// Check output
			if !output.Success {
				t.Error("Execute() success = false, want true")
			}

			if output.Path != tt.relativePath {
				t.Errorf("Execute() path = %q, want %q", output.Path, tt.relativePath)
			}

			// Verify file was actually written
			actualFilePath := filepath.Join(workspaceDir, tt.relativePath)
			actualContent, err := os.ReadFile(actualFilePath)
			if err != nil {
				t.Errorf("failed to read written file: %v", err)
				return
			}

			if string(actualContent) != tt.content {
				t.Errorf("written content = %q, want %q", string(actualContent), tt.content)
			}
		})
	}
}

func TestFileReadWrite_Integration(t *testing.T) {
	// Create a temporary workspace directory
	workspaceDir, err := os.MkdirTemp("", "filetools-integration-*")
	if err != nil {
		t.Fatalf("failed to create workspace dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(workspaceDir)

	relativePath := "integration.txt"
	originalContent := "Original content"

	// Write content
	writeInput := FileWriteInput{
		Path:    relativePath,
		Content: originalContent,
	}

	writeOutput, err := executeFileWrite(workspaceDir, writeInput)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if !writeOutput.Success {
		t.Error("write failed")
	}

	// Read content back
	readInput := FileReadInput{Path: relativePath}

	readOutput, err := executeFileRead(workspaceDir, readInput)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if readOutput.Content != originalContent {
		t.Errorf("read content = %q, want %q", readOutput.Content, originalContent)
	}

	// Update content
	updatedContent := "Updated content"
	writeInput.Content = updatedContent

	writeOutput, err = executeFileWrite(workspaceDir, writeInput)
	if err != nil {
		t.Fatalf("failed to update file: %v", err)
	}

	if !writeOutput.Success {
		t.Error("update write failed")
	}

	// Read updated content
	readOutput, err = executeFileRead(workspaceDir, readInput)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}

	if readOutput.Content != updatedContent {
		t.Errorf("updated content = %q, want %q", readOutput.Content, updatedContent)
	}
}

func TestResolveWorkspacePath_Security(t *testing.T) {
	tests := []struct {
		name        string
		userPath    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid relative path",
			userPath: "file.txt",
			wantErr:  false,
		},
		{
			name:     "valid nested path",
			userPath: "subdir/file.txt",
			wantErr:  false,
		},
		{
			name:        "absolute path",
			userPath:    "/etc/passwd",
			wantErr:     true,
			errContains: "absolute paths are not allowed",
		},
		{
			name:        "path traversal with ..",
			userPath:    "../outside.txt",
			wantErr:     true,
			errContains: "path traversal detected",
		},
		{
			name:        "path traversal from subdirectory",
			userPath:    "subdir/../../outside.txt",
			wantErr:     true,
			errContains: "path traversal detected",
		},
		{
			name:     "multiple slashes",
			userPath: "subdir//file.txt",
			wantErr:  false,
		},
		{
			name:     "dot current directory",
			userPath: "./file.txt",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary workspace directory
			workspaceDir, err := os.MkdirTemp("", "workspace-security-*")
			if err != nil {
				t.Fatalf("failed to create workspace dir: %v", err)
			}
			defer func(path string) {
				_ = os.RemoveAll(path)
			}(workspaceDir)

			// Test resolveWorkspacePath
			resolvedPath, err := resolveWorkspacePath(workspaceDir, tt.userPath)

			// Check error expectations
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveWorkspacePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("resolveWorkspacePath() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			// If no error, verify the resolved path is within workspace
			if !tt.wantErr {
				absWorkspace, _ := filepath.Abs(workspaceDir)
				if !strings.HasPrefix(resolvedPath, absWorkspace) {
					t.Errorf("resolvedPath %q is not within workspace %q", resolvedPath, absWorkspace)
				}
			}
		})
	}
}

// TestFileReadTool_ToolCreation tests that the tool creation functions work correctly
func TestFileReadTool_ToolCreation(t *testing.T) {
	t.Run("default workspace", func(t *testing.T) {
		tool := FileReadTool()
		if tool == nil {
			t.Fatal("FileReadTool() returned nil")
		}
	})

	t.Run("custom workspace", func(t *testing.T) {
		workspaceDir, err := os.MkdirTemp("", "filetools-creation-*")
		if err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}
		defer func(path string) {
			_ = os.RemoveAll(path)
		}(workspaceDir)

		tool := NewFileReadToolWithWorkspace(workspaceDir)
		if tool == nil {
			t.Fatal("NewFileReadToolWithWorkspace() returned nil")
		}
	})
}

// TestFileWriteTool_ToolCreation tests that the tool creation functions work correctly
func TestFileWriteTool_ToolCreation(t *testing.T) {
	t.Run("default workspace", func(t *testing.T) {
		tool := FileWriteTool()
		if tool == nil {
			t.Fatal("FileWriteTool() returned nil")
		}
	})

	t.Run("custom workspace", func(t *testing.T) {
		workspaceDir, err := os.MkdirTemp("", "filetools-creation-*")
		if err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}
		defer func(path string) {
			_ = os.RemoveAll(path)
		}(workspaceDir)

		tool := NewFileWriteToolWithWorkspace(workspaceDir)
		if tool == nil {
			t.Fatal("NewFileWriteToolWithWorkspace() returned nil")
		}
	})
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
