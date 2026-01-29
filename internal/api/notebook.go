package api

// Notebook represents a SiYuan notebook
type Notebook struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Icon   string `json:"icon"`
	Sort   int    `json:"sort"`
	Closed bool   `json:"closed"`
}

// NotebooksResponse is the response from lsNotebooks
type NotebooksResponse struct {
	Notebooks []*Notebook `json:"notebooks"`
}

// ListNotebooks returns all notebooks in the workspace
func (c *Client) ListNotebooks() ([]*Notebook, error) {
	var resp NotebooksResponse
	if err := c.PostData("/api/notebook/lsNotebooks", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Notebooks, nil
}

// GetNotebookByName finds a notebook by its name
func (c *Client) GetNotebookByName(name string) (*Notebook, error) {
	notebooks, err := c.ListNotebooks()
	if err != nil {
		return nil, err
	}
	for _, nb := range notebooks {
		if nb.Name == name {
			return nb, nil
		}
	}
	return nil, nil
}

// GetNotebookByID finds a notebook by its ID
func (c *Client) GetNotebookByID(id string) (*Notebook, error) {
	notebooks, err := c.ListNotebooks()
	if err != nil {
		return nil, err
	}
	for _, nb := range notebooks {
		if nb.ID == id {
			return nb, nil
		}
	}
	return nil, nil
}
