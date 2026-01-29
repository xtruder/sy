package api

import "encoding/json"

// Block represents a SiYuan block
type Block struct {
	ID       string          `json:"id"`
	ParentID string          `json:"parentID"`
	RootID   string          `json:"rootID"`
	Hash     string          `json:"hash"`
	Box      string          `json:"box"`   // Notebook ID
	Path     string          `json:"path"`  // Storage path
	HPath    string          `json:"hpath"` // Human-readable path
	Name     string          `json:"name"`  // Document name (for doc blocks)
	Alias    string          `json:"alias"`
	Memo     string          `json:"memo"`
	Tag      string          `json:"tag"`
	Content  string          `json:"content"`  // Plain text content
	FContent string          `json:"fcontent"` // Full content
	Markdown string          `json:"markdown"`
	Length   int             `json:"length"`
	Type     string          `json:"type"` // p, h, d, l, i, c, etc.
	SubType  string          `json:"subType"`
	IAL      json.RawMessage `json:"ial"` // Inline Attribute List (can be string or object)
	Sort     int             `json:"sort"`
	Created  string          `json:"created"`
	Updated  string          `json:"updated"`
}

// ChildBlock is a simplified block representation from getChildBlocks
type ChildBlock struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	SubType string `json:"subType"`
}

// GetChildBlocksRequest is the request for getChildBlocks
type GetChildBlocksRequest struct {
	ID string `json:"id"`
}

// GetChildBlocks returns the direct children of a block (typically a document)
func (c *Client) GetChildBlocks(id string) ([]*ChildBlock, error) {
	req := GetChildBlocksRequest{ID: id}
	var children []*ChildBlock
	if err := c.PostData("/api/block/getChildBlocks", req, &children); err != nil {
		return nil, err
	}
	return children, nil
}

// KramdownResponse is the response from getBlockKramdown
type KramdownResponse struct {
	ID       string `json:"id"`
	Kramdown string `json:"kramdown"`
}

// GetBlockKramdownRequest is the request for getBlockKramdown
type GetBlockKramdownRequest struct {
	ID string `json:"id"`
}

// GetBlockKramdown exports a block as kramdown (includes nested children with IAL)
func (c *Client) GetBlockKramdown(id string) (string, error) {
	req := GetBlockKramdownRequest{ID: id}
	var resp KramdownResponse
	if err := c.PostData("/api/block/getBlockKramdown", req, &resp); err != nil {
		return "", err
	}
	return resp.Kramdown, nil
}

// UpdateBlockRequest is the request for updateBlock
type UpdateBlockRequest struct {
	DataType string `json:"dataType"` // "markdown" or "dom"
	Data     string `json:"data"`
	ID       string `json:"id"`
}

// UpdateBlockTransaction is a transaction from updateBlock (same structure as insert)
type UpdateBlockTransaction struct {
	DoOperations []struct {
		Action   string `json:"action"`
		ID       string `json:"id"`
		ParentID string `json:"parentID"`
	} `json:"doOperations"`
}

// UpdateBlock updates a block's content
func (c *Client) UpdateBlock(id, markdown string) error {
	req := UpdateBlockRequest{
		DataType: "markdown",
		Data:     markdown,
		ID:       id,
	}
	var resp []UpdateBlockTransaction
	return c.PostData("/api/block/updateBlock", req, &resp)
}

// InsertBlockRequest is the request for insertBlock
type InsertBlockRequest struct {
	DataType   string `json:"dataType"` // "markdown" or "dom"
	Data       string `json:"data"`
	PreviousID string `json:"previousID"` // Block after which to insert
	ParentID   string `json:"parentID"`   // Parent block (for children)
}

// InsertBlockOperation represents a single insert operation
type InsertBlockOperation struct {
	Action string `json:"action"`
	ID     string `json:"id"`
}

// InsertBlockTransaction represents a transaction from insertBlock
type InsertBlockTransaction struct {
	DoOperations []InsertBlockOperation `json:"doOperations"`
}

// InsertBlock inserts a new block after the specified block
func (c *Client) InsertBlockAfter(previousID, markdown string) (string, error) {
	req := InsertBlockRequest{
		DataType:   "markdown",
		Data:       markdown,
		PreviousID: previousID,
	}
	var resp []InsertBlockTransaction
	if err := c.PostData("/api/block/insertBlock", req, &resp); err != nil {
		return "", err
	}
	if len(resp) > 0 && len(resp[0].DoOperations) > 0 {
		return resp[0].DoOperations[0].ID, nil
	}
	return "", nil
}

// InsertBlockAsChild inserts a new block as a child of the specified parent
func (c *Client) InsertBlockAsChild(parentID, markdown string) (string, error) {
	req := InsertBlockRequest{
		DataType: "markdown",
		Data:     markdown,
		ParentID: parentID,
	}
	var resp []InsertBlockTransaction
	if err := c.PostData("/api/block/insertBlock", req, &resp); err != nil {
		return "", err
	}
	if len(resp) > 0 && len(resp[0].DoOperations) > 0 {
		return resp[0].DoOperations[0].ID, nil
	}
	return "", nil
}

// DeleteBlockRequest is the request for deleteBlock
type DeleteBlockRequest struct {
	ID string `json:"id"`
}

// DeleteBlock deletes a block by ID
func (c *Client) DeleteBlock(id string) error {
	req := DeleteBlockRequest{ID: id}
	return c.PostData("/api/block/deleteBlock", req, nil)
}

// GetBlockInfo gets basic info about a block
type GetBlockInfoRequest struct {
	ID string `json:"id"`
}

type BlockInfo struct {
	Box       string `json:"box"`
	Path      string `json:"path"`
	RootID    string `json:"rootID"`
	RootTitle string `json:"rootTitle"`
	RootIcon  string `json:"rootIcon"`
}

func (c *Client) GetBlockInfo(id string) (*BlockInfo, error) {
	req := GetBlockInfoRequest{ID: id}
	var resp BlockInfo
	if err := c.PostData("/api/block/getBlockInfo", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetBlockTitle returns a suitable title/anchor text for a block reference.
// For documents, returns the document title.
// For other blocks, returns a truncated version of the content.
func (c *Client) GetBlockTitle(id string) (string, error) {
	// First try to get block info (works for all blocks, gives us root title)
	info, err := c.GetBlockInfo(id)
	if err != nil {
		return "", err
	}

	// If the block IS the document (rootID == id), use root title
	if info.RootID == id {
		return info.RootTitle, nil
	}

	// For non-document blocks, query the blocks table for content
	query := "SELECT content FROM blocks WHERE id = '" + id + "' LIMIT 1"
	results, err := c.ExecuteSQL(query)
	if err != nil {
		// Fallback to root title if query fails
		return info.RootTitle, nil
	}

	if len(results) > 0 {
		if content, ok := results[0]["content"].(string); ok && content != "" {
			// Truncate long content
			if len(content) > 50 {
				content = content[:47] + "..."
			}
			return content, nil
		}
	}

	// Fallback to root title
	return info.RootTitle, nil
}
