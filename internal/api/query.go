package api

import "encoding/json"

// SQLRequest is the request for SQL query
type SQLRequest struct {
	Stmt string `json:"stmt"`
}

// ExecuteSQL executes a SQL query and returns the raw result
// The result is a slice of maps representing rows
func (c *Client) ExecuteSQL(stmt string) ([]map[string]interface{}, error) {
	req := SQLRequest{Stmt: stmt}

	resp, err := c.Post("/api/query/sql", req)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// FullTextSearchRequest is the request for full-text search
type FullTextSearchRequest struct {
	Query   string       `json:"query"`
	Types   *SearchTypes `json:"types,omitempty"`
	Method  int          `json:"method"`  // 0: keyword, 1: query syntax, 2: SQL, 3: regex
	GroupBy int          `json:"groupBy"` // 0: no grouping, 1: by document
}

// SearchTypes specifies which block types to search
type SearchTypes struct {
	Paragraph  bool `json:"paragraph"`
	Heading    bool `json:"heading"`
	MathBlock  bool `json:"mathBlock"`
	Table      bool `json:"table"`
	CodeBlock  bool `json:"codeBlock"`
	HTMLBlock  bool `json:"htmlBlock"`
	EmbedBlock bool `json:"embedBlock"`
}

// SearchResult is a single search result
type SearchResult struct {
	Block *Block `json:"block"`
	Path  string `json:"path"`
	HPath string `json:"hPath"`
}

// FullTextSearchResponse is the response from fullTextSearchBlock
type FullTextSearchResponse struct {
	Blocks            []*Block `json:"blocks"`
	MatchedRootIDs    []string `json:"matchedRootIDs"`
	MatchedBlockCount int      `json:"matchedBlockCount"`
}

// FullTextSearch performs a full-text search
func (c *Client) FullTextSearch(query string, types *SearchTypes) (*FullTextSearchResponse, error) {
	req := FullTextSearchRequest{
		Query:   query,
		Types:   types,
		Method:  0, // Keyword search
		GroupBy: 0,
	}

	var resp FullTextSearchResponse
	if err := c.PostData("/api/search/fullTextSearchBlock", req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
