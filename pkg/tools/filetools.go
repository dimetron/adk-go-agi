package tools

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// DefaultWorkspaceDir is the default directory for file operations
const DefaultWorkspaceDir = "./workspace"

// MaxFileSize is the maximum file size allowed for read/write operations (10MB)
const MaxFileSize = 10 * 1024 * 1024

// FileOperationTimeout is the timeout for file I/O operations
const FileOperationTimeout = 30 * time.Second

// FileReadInput defines the input parameters for the fileRead tool
type FileReadInput struct {
	// Path is the relative path to the file to read (within the workspace directory)
	Path string `json:"path"`
}

// FileReadOutput defines the output structure for the fileRead tool
type FileReadOutput struct {
	// Content is the content of the file
	Content string `json:"content,omitempty"`
	// Path is the path of the file that was read
	Path string `json:"path,omitempty"`
	// Error contains the error message if the operation failed
	Error string `json:"error,omitempty"`
}

// FileWriteInput defines the input parameters for the fileWrite tool
type FileWriteInput struct {
	// Path is the relative path to the file to write (within the workspace directory)
	Path string `json:"path"`
	// Content is the content to write to the file
	Content string `json:"content"`
}

// FileWriteOutput defines the output structure for the fileWrite tool
type FileWriteOutput struct {
	// Path is the path of the file that was written
	Path string `json:"path,omitempty"`
	// Success indicates whether the write operation was successful
	Success bool `json:"success"`
	// Error contains the error message if the operation failed
	Error string `json:"error,omitempty"`
}

// FileReadTool creates a new fileRead tool that reads the content of a file within the workspace directory
func FileReadTool() tool.Tool {
	return NewFileReadToolWithWorkspace(DefaultWorkspaceDir)
}

// NewFileReadToolWithWorkspace creates a new fileRead tool with a custom workspace directory
func NewFileReadToolWithWorkspace(workspaceDir string) tool.Tool {
	t, err := functiontool.New(
		functiontool.Config{
			Name:        "fileRead",
			Description: "Read the content of a file from the workspace directory. All paths are relative to the workspace.",
		},
		func(ctx tool.Context, input FileReadInput) *FileReadOutput {
			start := time.Now()
			slog.Info("Starting file read operation",
				"path", input.Path,
				"workspace", workspaceDir)

			// Validate and resolve the path within workspace
			resolvedPath, err := resolveWorkspacePath(workspaceDir, input.Path)
			if err != nil {
				slog.Error("Failed to resolve path",
					"path", input.Path,
					"error", err)
				return &FileReadOutput{
					Error: fmt.Sprintf("Failed to resolve path: %v", err),
				}
			}

			// Check file size before reading to prevent reading huge files
			info, err := os.Stat(resolvedPath)
			if err != nil {
				slog.Error("Failed to stat file",
					"path", input.Path,
					"resolved_path", resolvedPath,
					"error", err)
				return &FileReadOutput{
					Error: fmt.Sprintf("Failed to stat file %s: %v", input.Path, err),
				}
			}

			if info.Size() > MaxFileSize {
				slog.Warn("File too large",
					"path", input.Path,
					"size_bytes", info.Size(),
					"max_size_bytes", MaxFileSize)
				return &FileReadOutput{
					Error: fmt.Sprintf("File too large: %d bytes (max %d bytes)", info.Size(), MaxFileSize),
				}
			}

			// Use context with timeout for file read operation
			readCtx, cancel := context.WithTimeout(context.Background(), FileOperationTimeout)
			defer cancel()

			// Perform file read with timeout
			done := make(chan struct{})
			var content []byte
			var readErr error

			go func() {
				content, readErr = os.ReadFile(resolvedPath)
				close(done)
			}()

			select {
			case <-done:
				if readErr != nil {
					slog.Error("Failed to read file",
						"path", input.Path,
						"error", readErr,
						"duration_ms", time.Since(start).Milliseconds())
					return &FileReadOutput{
						Error: fmt.Sprintf("Failed to read file %s: %v", input.Path, readErr),
					}
				}

				slog.Info("File read completed successfully",
					"path", input.Path,
					"size_bytes", len(content),
					"duration_ms", time.Since(start).Milliseconds())

				return &FileReadOutput{
					Content: string(content),
					Path:    input.Path,
				}
			case <-readCtx.Done():
				slog.Error("File read operation timed out",
					"path", input.Path,
					"timeout", FileOperationTimeout)
				return &FileReadOutput{
					Error: fmt.Sprintf("File read timeout exceeded (%v)", FileOperationTimeout),
				}
			}
		},
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create fileRead tool: %v", err))
	}
	return t
}

// FileWriteTool creates a new fileWrite tool that writes content to a file within the workspace directory
func FileWriteTool() tool.Tool {
	return NewFileWriteToolWithWorkspace(DefaultWorkspaceDir)
}

// NewFileWriteToolWithWorkspace creates a new fileWrite tool with a custom workspace directory
func NewFileWriteToolWithWorkspace(workspaceDir string) tool.Tool {
	t, err := functiontool.New(
		functiontool.Config{
			Name:        "fileWrite",
			Description: "Write content to a file in the workspace directory. Creates the file if it doesn't exist, or overwrites it if it does. All paths are relative to the workspace.",
		},
		func(ctx tool.Context, input FileWriteInput) *FileWriteOutput {
			start := time.Now()
			slog.Info("Starting file write operation",
				"path", input.Path,
				"content_size_bytes", len(input.Content),
				"workspace", workspaceDir)

			// Check content size before writing
			if len(input.Content) > MaxFileSize {
				slog.Warn("Content too large",
					"path", input.Path,
					"size_bytes", len(input.Content),
					"max_size_bytes", MaxFileSize)
				return &FileWriteOutput{
					Success: false,
					Error:   fmt.Sprintf("Content too large: %d bytes (max %d bytes)", len(input.Content), MaxFileSize),
				}
			}

			// Validate and resolve the path within workspace
			resolvedPath, err := resolveWorkspacePath(workspaceDir, input.Path)
			if err != nil {
				slog.Error("Failed to resolve path",
					"path", input.Path,
					"error", err)
				return &FileWriteOutput{
					Success: false,
					Error:   fmt.Sprintf("Failed to resolve path: %v", err),
				}
			}

			// Ensure the directory exists
			dir := filepath.Dir(resolvedPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				slog.Error("Failed to create directory",
					"path", input.Path,
					"directory", dir,
					"error", err)
				return &FileWriteOutput{
					Success: false,
					Error:   fmt.Sprintf("Failed to create directory for %s: %v", input.Path, err),
				}
			}

			// Use context with timeout for file write operation
			writeCtx, cancel := context.WithTimeout(context.Background(), FileOperationTimeout)
			defer cancel()

			// Perform file write with timeout
			done := make(chan struct{})
			var writeErr error

			go func() {
				writeErr = os.WriteFile(resolvedPath, []byte(input.Content), 0644)
				close(done)
			}()

			select {
			case <-done:
				if writeErr != nil {
					slog.Error("Failed to write file",
						"path", input.Path,
						"error", writeErr,
						"duration_ms", time.Since(start).Milliseconds())
					return &FileWriteOutput{
						Success: false,
						Error:   fmt.Sprintf("Failed to write file %s: %v", input.Path, writeErr),
					}
				}

				slog.Info("File write completed successfully",
					"path", input.Path,
					"size_bytes", len(input.Content),
					"duration_ms", time.Since(start).Milliseconds())

				return &FileWriteOutput{
					Path:    input.Path,
					Success: true,
				}
			case <-writeCtx.Done():
				slog.Error("File write operation timed out",
					"path", input.Path,
					"timeout", FileOperationTimeout)
				return &FileWriteOutput{
					Success: false,
					Error:   fmt.Sprintf("File write timeout exceeded (%v)", FileOperationTimeout),
				}
			}
		},
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create fileWrite tool: %v", err))
	}
	return t
}

// resolveWorkspacePath validates and resolves a user-provided path within the workspace directory.
// It prevents directory traversal attacks and ensures all operations stay within the workspace.
func resolveWorkspacePath(workspaceDir, userPath string) (string, error) {
	// Clean the user path to remove any ".." or other traversal attempts
	cleanUserPath := filepath.Clean(userPath)

	// Prevent absolute paths
	if filepath.IsAbs(cleanUserPath) {
		return "", fmt.Errorf("absolute paths are not allowed: %s", userPath)
	}

	// Get absolute path of workspace
	absWorkspace, err := filepath.Abs(workspaceDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace directory: %w", err)
	}

	// Ensure workspace directory exists
	if err := os.MkdirAll(absWorkspace, 0755); err != nil {
		return "", fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Join workspace with user path
	fullPath := filepath.Join(absWorkspace, cleanUserPath)

	// Get absolute path of the result
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve file path: %w", err)
	}

	// Ensure the resolved path is still within the workspace
	// This prevents directory traversal attacks
	if !strings.HasPrefix(absFullPath, absWorkspace+string(filepath.Separator)) &&
		absFullPath != absWorkspace {
		return "", fmt.Errorf("path traversal detected: %s escapes workspace directory", userPath)
	}

	return absFullPath, nil
}
