package api

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/asergenalkan/serverpanel/internal/models"
	"github.com/gofiber/fiber/v2"
)

// FileInfo represents file/directory information
type FileInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	IsDir       bool      `json:"is_dir"`
	Size        int64     `json:"size"`
	Modified    time.Time `json:"modified"`
	Permissions string    `json:"permissions"`
	Extension   string    `json:"extension,omitempty"`
}

// getUserFromContext gets user info from context (set by auth middleware)
func getUserFromContext(c *fiber.Ctx) (username string, role string) {
	username, _ = c.Locals("username").(string)
	role, _ = c.Locals("role").(string)
	return
}

// validatePath ensures the path is within the user's allowed directory
func (h *Handler) validatePath(basePath, requestedPath string) (string, error) {
	// Clean and join the paths
	fullPath := filepath.Clean(filepath.Join(basePath, requestedPath))

	// Ensure the path is within the base directory (prevent path traversal)
	if !strings.HasPrefix(fullPath, basePath) {
		return "", fmt.Errorf("access denied: path outside home directory")
	}

	return fullPath, nil
}

// ListFiles returns directory contents
func (h *Handler) ListFiles(c *fiber.Ctx) error {
	username, role := getUserFromContext(c)
	path := c.Query("path", "/")

	var basePath string
	if role == models.RoleAdmin {
		// Admin: path is relative to HomeBaseDir
		basePath = h.cfg.HomeBaseDir
	} else {
		// User: path is relative to their home directory
		basePath = filepath.Join(h.cfg.HomeBaseDir, username)
	}

	fullPath, err := h.validatePath(basePath, path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	// Check if path exists
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Path not found"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	if !info.IsDir() {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Path is not a directory"})
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		relativePath := filepath.Join(path, entry.Name())
		ext := ""
		if !entry.IsDir() {
			ext = strings.TrimPrefix(filepath.Ext(entry.Name()), ".")
		}

		files = append(files, FileInfo{
			Name:        entry.Name(),
			Path:        relativePath,
			IsDir:       entry.IsDir(),
			Size:        info.Size(),
			Modified:    info.ModTime(),
			Permissions: info.Mode().String(),
			Extension:   ext,
		})
	}

	// Sort: directories first, then by name
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	// Calculate current path relative to base
	currentPath := path
	if currentPath == "" || currentPath == "/" {
		currentPath = "/"
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"path":  currentPath,
			"files": files,
		},
	})
}

// ReadFile returns file content
func (h *Handler) ReadFile(c *fiber.Ctx) error {
	username, role := getUserFromContext(c)
	path := c.Query("path")

	if path == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Path is required"})
	}

	var basePath string
	if role == models.RoleAdmin {
		basePath = h.cfg.HomeBaseDir
	} else {
		basePath = filepath.Join(h.cfg.HomeBaseDir, username)
	}

	fullPath, err := h.validatePath(basePath, path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	// Check if file exists and is not a directory
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "File not found"})
	}
	if info.IsDir() {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Cannot read directory"})
	}

	// Limit file size for reading (10MB)
	if info.Size() > 10*1024*1024 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "File too large to read (max 10MB)"})
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"path":    path,
			"content": string(content),
			"size":    info.Size(),
		},
	})
}

// WriteFile creates or updates a file
func (h *Handler) WriteFile(c *fiber.Ctx) error {
	username, role := getUserFromContext(c)

	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Invalid request"})
	}

	if req.Path == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Path is required"})
	}

	var basePath string
	if role == models.RoleAdmin {
		basePath = h.cfg.HomeBaseDir
	} else {
		basePath = filepath.Join(h.cfg.HomeBaseDir, username)
	}

	fullPath, err := h.validatePath(basePath, req.Path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	// Create parent directories if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	log.Printf("📝 File written: %s by %s", req.Path, username)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "File saved successfully",
	})
}

// CreateDirectory creates a new directory
func (h *Handler) CreateDirectory(c *fiber.Ctx) error {
	username, role := getUserFromContext(c)

	var req struct {
		Path string `json:"path"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Invalid request"})
	}

	var basePath string
	if role == models.RoleAdmin {
		basePath = h.cfg.HomeBaseDir
	} else {
		basePath = filepath.Join(h.cfg.HomeBaseDir, username)
	}

	fullPath, err := h.validatePath(basePath, req.Path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	log.Printf("📁 Directory created: %s by %s", req.Path, username)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Directory created successfully",
	})
}

// DeleteFiles deletes files or directories
func (h *Handler) DeleteFiles(c *fiber.Ctx) error {
	username, role := getUserFromContext(c)

	var req struct {
		Paths []string `json:"paths"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Invalid request"})
	}

	if len(req.Paths) == 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "No paths specified"})
	}

	var basePath string
	if role == models.RoleAdmin {
		basePath = h.cfg.HomeBaseDir
	} else {
		basePath = filepath.Join(h.cfg.HomeBaseDir, username)
	}

	var errors []string
	for _, path := range req.Paths {
		fullPath, err := h.validatePath(basePath, path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", path, err))
			continue
		}

		// Prevent deleting root directories
		if fullPath == basePath || fullPath == h.cfg.HomeBaseDir {
			errors = append(errors, fmt.Sprintf("%s: cannot delete root directory", path))
			continue
		}

		if err := os.RemoveAll(fullPath); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", path, err))
		}
	}

	log.Printf("🗑️ Files deleted by %s: %v", username, req.Paths)

	if len(errors) > 0 {
		return c.JSON(fiber.Map{
			"success": false,
			"error":   strings.Join(errors, "; "),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Files deleted successfully",
	})
}

// RenameFile renames a file or directory
func (h *Handler) RenameFile(c *fiber.Ctx) error {
	username, role := getUserFromContext(c)

	var req struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Invalid request"})
	}

	var basePath string
	if role == models.RoleAdmin {
		basePath = h.cfg.HomeBaseDir
	} else {
		basePath = filepath.Join(h.cfg.HomeBaseDir, username)
	}

	oldFullPath, err := h.validatePath(basePath, req.OldPath)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	newFullPath, err := h.validatePath(basePath, req.NewPath)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	if err := os.Rename(oldFullPath, newFullPath); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	log.Printf("📝 File renamed by %s: %s -> %s", username, req.OldPath, req.NewPath)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "File renamed successfully",
	})
}

// CopyFiles copies files or directories
func (h *Handler) CopyFiles(c *fiber.Ctx) error {
	username, role := getUserFromContext(c)

	var req struct {
		Sources     []string `json:"sources"`
		Destination string   `json:"destination"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Invalid request"})
	}

	var basePath string
	if role == models.RoleAdmin {
		basePath = h.cfg.HomeBaseDir
	} else {
		basePath = filepath.Join(h.cfg.HomeBaseDir, username)
	}

	destPath, err := h.validatePath(basePath, req.Destination)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	var errors []string
	for _, src := range req.Sources {
		srcPath, err := h.validatePath(basePath, src)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", src, err))
			continue
		}

		newPath := filepath.Join(destPath, filepath.Base(srcPath))
		if err := copyPath(srcPath, newPath); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", src, err))
		}
	}

	if len(errors) > 0 {
		return c.JSON(fiber.Map{"success": false, "error": strings.Join(errors, "; ")})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Files copied successfully"})
}

// MoveFiles moves files or directories
func (h *Handler) MoveFiles(c *fiber.Ctx) error {
	username, role := getUserFromContext(c)

	var req struct {
		Sources     []string `json:"sources"`
		Destination string   `json:"destination"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Invalid request"})
	}

	var basePath string
	if role == models.RoleAdmin {
		basePath = h.cfg.HomeBaseDir
	} else {
		basePath = filepath.Join(h.cfg.HomeBaseDir, username)
	}

	destPath, err := h.validatePath(basePath, req.Destination)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	var errors []string
	for _, src := range req.Sources {
		srcPath, err := h.validatePath(basePath, src)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", src, err))
			continue
		}

		newPath := filepath.Join(destPath, filepath.Base(srcPath))
		if err := os.Rename(srcPath, newPath); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", src, err))
		}
	}

	if len(errors) > 0 {
		return c.JSON(fiber.Map{"success": false, "error": strings.Join(errors, "; ")})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Files moved successfully"})
}

// UploadFiles handles file uploads
func (h *Handler) UploadFiles(c *fiber.Ctx) error {
	username, role := getUserFromContext(c)
	path := c.FormValue("path", "/")

	var basePath string
	if role == models.RoleAdmin {
		basePath = h.cfg.HomeBaseDir
	} else {
		basePath = filepath.Join(h.cfg.HomeBaseDir, username)
	}

	destPath, err := h.validatePath(basePath, path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	// Get all files from the multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Failed to parse form"})
	}

	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "No files uploaded"})
	}

	var uploaded []string
	var errors []string

	for _, file := range files {
		filePath := filepath.Join(destPath, file.Filename)

		if err := c.SaveFile(file, filePath); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", file.Filename, err))
		} else {
			uploaded = append(uploaded, file.Filename)
		}
	}

	log.Printf("📤 Files uploaded by %s to %s: %v", username, path, uploaded)

	if len(errors) > 0 {
		return c.JSON(fiber.Map{
			"success":  len(uploaded) > 0,
			"uploaded": uploaded,
			"errors":   errors,
		})
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"message":  fmt.Sprintf("%d file(s) uploaded successfully", len(uploaded)),
		"uploaded": uploaded,
	})
}

// DownloadFile serves a file for download
func (h *Handler) DownloadFile(c *fiber.Ctx) error {
	username, role := getUserFromContext(c)
	path := c.Query("path")

	if path == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Path is required"})
	}

	var basePath string
	if role == models.RoleAdmin {
		basePath = h.cfg.HomeBaseDir
	} else {
		basePath = filepath.Join(h.cfg.HomeBaseDir, username)
	}

	fullPath, err := h.validatePath(basePath, path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "File not found"})
	}
	if info.IsDir() {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Cannot download directory"})
	}

	// Set content type
	ext := filepath.Ext(fullPath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	c.Set("Content-Type", mimeType)
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(fullPath)))

	return c.SendFile(fullPath)
}

// CompressFiles creates a zip archive
func (h *Handler) CompressFiles(c *fiber.Ctx) error {
	username, role := getUserFromContext(c)

	var req struct {
		Paths       []string `json:"paths"`
		Destination string   `json:"destination"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Invalid request"})
	}

	var basePath string
	if role == models.RoleAdmin {
		basePath = h.cfg.HomeBaseDir
	} else {
		basePath = filepath.Join(h.cfg.HomeBaseDir, username)
	}

	destPath, err := h.validatePath(basePath, req.Destination)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	// Ensure destination ends with .zip
	if !strings.HasSuffix(destPath, ".zip") {
		destPath += ".zip"
	}

	// Create zip file
	zipFile, err := os.Create(destPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, path := range req.Paths {
		srcPath, err := h.validatePath(basePath, path)
		if err != nil {
			continue
		}

		err = addToZip(zipWriter, srcPath, filepath.Base(srcPath))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
		}
	}

	log.Printf("📦 Files compressed by %s: %v -> %s", username, req.Paths, req.Destination)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Files compressed successfully",
		"path":    req.Destination,
	})
}

// ExtractFiles extracts a zip archive
func (h *Handler) ExtractFiles(c *fiber.Ctx) error {
	username, role := getUserFromContext(c)

	var req struct {
		Path        string `json:"path"`
		Destination string `json:"destination"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Invalid request"})
	}

	var basePath string
	if role == models.RoleAdmin {
		basePath = h.cfg.HomeBaseDir
	} else {
		basePath = filepath.Join(h.cfg.HomeBaseDir, username)
	}

	zipPath, err := h.validatePath(basePath, req.Path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	destPath, err := h.validatePath(basePath, req.Destination)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	// Open zip file
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	defer r.Close()

	for _, f := range r.File {
		fPath := filepath.Join(destPath, f.Name)

		// Prevent zip slip
		if !strings.HasPrefix(fPath, destPath) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fPath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fPath), 0755); err != nil {
			continue
		}

		outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			continue
		}

		io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
	}

	log.Printf("📦 Files extracted by %s: %s -> %s", username, req.Path, req.Destination)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Files extracted successfully",
	})
}

// GetFileInfo returns detailed file information
func (h *Handler) GetFileInfo(c *fiber.Ctx) error {
	username, role := getUserFromContext(c)
	path := c.Query("path")

	if path == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Path is required"})
	}

	var basePath string
	if role == models.RoleAdmin {
		basePath = h.cfg.HomeBaseDir
	} else {
		basePath = filepath.Join(h.cfg.HomeBaseDir, username)
	}

	fullPath, err := h.validatePath(basePath, path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "File not found"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"name":        info.Name(),
			"path":        path,
			"size":        info.Size(),
			"is_dir":      info.IsDir(),
			"modified":    info.ModTime(),
			"permissions": info.Mode().String(),
		},
	})
}

// Helper functions

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if err := copyPath(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

func addToZip(w *zip.Writer, path, base string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			err := addToZip(w, filepath.Join(path, entry.Name()), filepath.Join(base, entry.Name()))
			if err != nil {
				return err
			}
		}
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = base
	header.Method = zip.Deflate

	writer, err := w.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}
