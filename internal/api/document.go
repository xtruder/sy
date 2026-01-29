package api

import "strings"

// Document represents a SiYuan document in the file tree
type Document struct {
	ID           string `json:"id"`           // Block ID of the document
	Name         string `json:"name"`         // Title of the document (includes .sy suffix)
	Icon         string `json:"icon"`         // Emoji icon
	SubType      string `json:"subType"`      // Document subtype
	Count        int    `json:"count"`        // Number of content children (unreliable)
	SubFileCount int    `json:"subFileCount"` // Number of sub-documents
}

// Title returns the document title without the .sy suffix
func (d *Document) Title() string {
	return strings.TrimSuffix(d.Name, ".sy")
}

// ListDocsByPathRequest is the request for listDocsByPath
type ListDocsByPathRequest struct {
	Notebook string `json:"notebook"` // Notebook ID
	Path     string `json:"path"`     // Path within notebook (e.g., "/", "/Folder")
}

// ListDocsByPathResponse is the response from listDocsByPath
type ListDocsByPathResponse struct {
	Files []*Document `json:"files"`
}

// ListDocsByPath lists documents in a notebook at a specific path
func (c *Client) ListDocsByPath(notebookID, path string) ([]*Document, error) {
	req := ListDocsByPathRequest{
		Notebook: notebookID,
		Path:     path,
	}
	var resp ListDocsByPathResponse
	if err := c.PostData("/api/filetree/listDocsByPath", req, &resp); err != nil {
		return nil, err
	}
	return resp.Files, nil
}

// DocumentInfo contains detailed information about a document
type DocumentInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	RefCount int               `json:"refCount"`
	SubCount int               `json:"subCount"`
	RefIDs   []string          `json:"refIDs"`
	IAL      map[string]string `json:"ial"`
}

// GetDocInfoRequest is the request for getDocInfo
type GetDocInfoRequest struct {
	ID string `json:"id"`
}

// GetDocInfo gets detailed info about a document
func (c *Client) GetDocInfo(id string) (*DocumentInfo, error) {
	req := GetDocInfoRequest{ID: id}
	var resp DocumentInfo
	if err := c.PostData("/api/block/getDocInfo", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// HPathResponse is the response from getHPathByID
type HPathResponse struct {
	HPath string `json:"hPath"`
}

// GetHPathByID gets the human-readable path for a document
func (c *Client) GetHPathByID(id string) (string, error) {
	req := map[string]string{"id": id}
	var hpath string
	if err := c.PostData("/api/filetree/getHPathByID", req, &hpath); err != nil {
		return "", err
	}
	return hpath, nil
}

// GetHPathByPath gets the human-readable path from storage path
func (c *Client) GetHPathByPath(notebookID, path string) (string, error) {
	req := map[string]string{
		"notebook": notebookID,
		"path":     path,
	}
	var hpath string
	if err := c.PostData("/api/filetree/getHPathByPath", req, &hpath); err != nil {
		return "", err
	}
	return hpath, nil
}

// CreateDocWithMdRequest is the request for createDocWithMd
type CreateDocWithMdRequest struct {
	Notebook string `json:"notebook"` // Notebook ID
	Path     string `json:"path"`     // Path within notebook (e.g., "/Folder/DocTitle")
	Markdown string `json:"markdown"` // Markdown content
}

// CreateDocWithMd creates a new document with markdown content
// Returns the new document's ID
func (c *Client) CreateDocWithMd(notebookID, path, markdown string) (string, error) {
	req := CreateDocWithMdRequest{
		Notebook: notebookID,
		Path:     path,
		Markdown: markdown,
	}
	var docID string
	if err := c.PostData("/api/filetree/createDocWithMd", req, &docID); err != nil {
		return "", err
	}
	return docID, nil
}

// RemoveDocRequest is the request for removeDoc
type RemoveDocRequest struct {
	Notebook string `json:"notebook"` // Notebook ID
	Path     string `json:"path"`     // Storage path of the document
}

// RemoveDoc deletes a document
func (c *Client) RemoveDoc(notebookID, path string) error {
	req := RemoveDocRequest{
		Notebook: notebookID,
		Path:     path,
	}
	return c.PostData("/api/filetree/removeDoc", req, nil)
}

// MoveDocsRequest is the request for moveDocs (legacy path-based API)
type MoveDocsRequest struct {
	FromPaths  []string `json:"fromPaths"`  // Source paths
	ToNotebook string   `json:"toNotebook"` // Destination notebook ID
	ToPath     string   `json:"toPath"`     // Destination path
}

// MoveDocs moves documents to a new location (legacy path-based API)
func (c *Client) MoveDocs(fromPaths []string, toNotebook, toPath string) error {
	req := MoveDocsRequest{
		FromPaths:  fromPaths,
		ToNotebook: toNotebook,
		ToPath:     toPath,
	}
	return c.PostData("/api/filetree/moveDocs", req, nil)
}

// MoveDocsByIDRequest is the request for moveDocsByID
type MoveDocsByIDRequest struct {
	FromIDs []string `json:"fromIDs"` // Source document IDs
	ToID    string   `json:"toID"`    // Destination parent document/notebook ID
}

// MoveDocsByID moves documents by their IDs to a new parent
func (c *Client) MoveDocsByID(fromIDs []string, toID string) error {
	req := MoveDocsByIDRequest{
		FromIDs: fromIDs,
		ToID:    toID,
	}
	return c.PostData("/api/filetree/moveDocsByID", req, nil)
}

// RenameDocRequest is the request for renameDoc
type RenameDocRequest struct {
	Notebook string `json:"notebook"` // Notebook ID
	Path     string `json:"path"`     // Storage path
	Title    string `json:"title"`    // New title
}

// RenameDoc renames a document
func (c *Client) RenameDoc(notebookID, path, title string) error {
	req := RenameDocRequest{
		Notebook: notebookID,
		Path:     path,
		Title:    title,
	}
	return c.PostData("/api/filetree/renameDoc", req, nil)
}
