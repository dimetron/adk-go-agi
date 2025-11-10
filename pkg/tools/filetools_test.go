package tools

import (
	"context"
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

			// Create the tool with custom workspace
			tool := NewFileReadToolWithWorkspace(workspaceDir)

			// Execute the tool
			ctx := context.Background()
			input := FileReadInput{Path: tt.relativePath}

			// Get the function from the tool
			fn, ok := tool.(interface {
				Execute(ctx context.Context, input FileReadInput) (*FileReadOutput, error)
			})
			if !ok {
				t.Fatal("tool does not implement expected interface")
			}

			output, err := fn.Execute(ctx, input)

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

			// Create the tool with custom workspace
			tool := NewFileWriteToolWithWorkspace(workspaceDir)

			// Execute the tool
			ctx := context.Background()
			input := FileWriteInput{
				Path:    tt.relativePath,
				Content: tt.content,
			}

			// Get the function from the tool
			fn, ok := tool.(interface {
				Execute(ctx context.Context, input FileWriteInput) (*FileWriteOutput, error)
			})
			if !ok {
				t.Fatal("tool does not implement expected interface")
			}

			output, err := fn.Execute(ctx, input)

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

	ctx := context.Background()
	relativePath := "integration.txt"
	originalContent := "Original content"

	// Write content
	writeTool := NewFileWriteToolWithWorkspace(workspaceDir)
	writeInput := FileWriteInput{
		Path:    relativePath,
		Content: originalContent,
	}

	writeFn, ok := writeTool.(interface {
		Execute(ctx context.Context, input FileWriteInput) (*FileWriteOutput, error)
	})
	if !ok {
		t.Fatal("write tool does not implement expected interface")
	}

	writeOutput, err := writeFn.Execute(ctx, writeInput)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if !writeOutput.Success {
		t.Error("write failed")
	}

	// Read content back
	readTool := NewFileReadToolWithWorkspace(workspaceDir)
	readInput := FileReadInput{Path: relativePath}

	readFn, ok := readTool.(interface {
		Execute(ctx context.Context, input FileReadInput) (*FileReadOutput, error)
	})
	if !ok {
		t.Fatal("read tool does not implement expected interface")
	}

	readOutput, err := readFn.Execute(ctx, readInput)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if readOutput.Content != originalContent {
		t.Errorf("read content = %q, want %q", readOutput.Content, originalContent)
	}

	// Update content
	updatedContent := "Updated content"
	writeInput.Content = updatedContent

	writeOutput, err = writeFn.Execute(ctx, writeInput)
	if err != nil {
		t.Fatalf("failed to update file: %v", err)
	}

	// Read updated content
	readOutput, err = readFn.Execute(ctx, readInput)
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
