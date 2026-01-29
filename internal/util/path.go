package util

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/xtruder/sy/internal/api"
)

// BlockIDPattern matches valid SiYuan block IDs: YYYYMMDDHHmmss-xxxxxxx
var BlockIDPattern = regexp.MustCompile(`^(\d{14})-([a-z0-9]{7})$`)

// IsBlockID checks if a string is a valid SiYuan block ID
func IsBlockID(s string) bool {
	return BlockIDPattern.MatchString(s)
}

// ResolveDocumentID resolves a document identifier to its block ID
// Input can be:
//   - A block ID: 20260118141851-qvxnjey
//   - A path within notebook: /Folder/Document or Folder/Document
func ResolveDocumentID(client *api.Client, notebook *api.Notebook, input string) (string, error) {
	// Check if it's already a block ID
	if IsBlockID(input) {
		return input, nil
	}

	if notebook == nil {
		return "", fmt.Errorf("no notebook selected")
	}

	// Normalize path
	docPath := input
	if !strings.HasPrefix(docPath, "/") {
		docPath = "/" + docPath
	}

	// Handle root path
	if docPath == "/" {
		return "", fmt.Errorf("path must include document name")
	}

	// Split path into parts and traverse
	pathParts := strings.Split(strings.Trim(docPath, "/"), "/")

	currentPath := "/"
	var currentDocID string

	for _, part := range pathParts {
		docs, err := client.ListDocsByPath(notebook.ID, currentPath)
		if err != nil {
			return "", fmt.Errorf("failed to list docs at %s: %w", currentPath, err)
		}

		found := false
		for _, doc := range docs {
			// Match by name or name without .sy extension
			docName := doc.Name
			docNameNoExt := strings.TrimSuffix(docName, ".sy")
			if docName == part || docNameNoExt == part {
				currentDocID = doc.ID
				if currentPath == "/" {
					currentPath = "/" + doc.ID
				} else {
					currentPath = currentPath + "/" + doc.ID
				}
				found = true
				break
			}
		}

		if !found {
			return "", fmt.Errorf("document not found: %s (looking for '%s' in %s)", input, part, currentPath)
		}
	}

	return currentDocID, nil
}

// GetDocumentMeta retrieves full metadata for a document by ID
func GetDocumentMeta(client *api.Client, docID string) (*DocumentMeta, error) {
	info, err := client.GetBlockInfo(docID)
	if err != nil {
		return nil, fmt.Errorf("failed to get block info: %w", err)
	}

	// Get human path
	hpath, err := client.GetHPathByID(docID)
	if err != nil {
		hpath = ""
	}

	// Get notebook name
	notebook, err := client.GetNotebookByID(info.Box)
	if err != nil {
		return nil, fmt.Errorf("failed to get notebook: %w", err)
	}

	notebookName := ""
	if notebook != nil {
		notebookName = notebook.Name
	}

	// Build full hpath with notebook name
	fullHPath := hpath
	if notebookName != "" && hpath != "" {
		fullHPath = "/" + notebookName + hpath
	}

	return &DocumentMeta{
		DocID:    docID,
		Notebook: info.Box,
		Path:     info.Path,
		HPath:    fullHPath,
		Title:    info.RootTitle,
	}, nil
}

// ResolvePathToStoragePath converts a display path to storage path
// Display: /Folder/SubFolder/Doc
// Storage: /docID1/docID2/docID3
func ResolvePathToStoragePath(client *api.Client, notebook *api.Notebook, displayPath string) (string, error) {
	if !strings.HasPrefix(displayPath, "/") {
		displayPath = "/" + displayPath
	}

	if displayPath == "/" {
		return "/", nil
	}

	pathParts := strings.Split(strings.Trim(displayPath, "/"), "/")
	currentPath := "/"

	for _, part := range pathParts {
		docs, err := client.ListDocsByPath(notebook.ID, currentPath)
		if err != nil {
			return "", fmt.Errorf("failed to list docs at %s: %w", currentPath, err)
		}

		found := false
		for _, doc := range docs {
			// Match by title (without .sy extension)
			if doc.Title() == part {
				if currentPath == "/" {
					currentPath = "/" + doc.ID
				} else {
					currentPath = currentPath + "/" + doc.ID
				}
				found = true
				break
			}
		}

		if !found {
			return "", fmt.Errorf("path not found: %s (looking for '%s')", displayPath, part)
		}
	}

	return currentPath, nil
}
