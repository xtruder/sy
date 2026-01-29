package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xtruder/sy/internal/api"
	"github.com/xtruder/sy/internal/config"
)

// completeDocumentPath provides completion for document paths
// It lists documents based on the current input prefix
func completeDocumentPath(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Don't complete if we already have an argument (for single-arg commands)
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return completePath(toComplete)
}

// completeMultipleDocumentPaths provides completion for commands that accept multiple document paths
func completeMultipleDocumentPaths(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completePath(toComplete)
}

// completePath does the actual path completion work
func completePath(toComplete string) ([]string, cobra.ShellCompDirective) {
	// Initialize client for completion
	client, notebook, err := initCompletionClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if notebook == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Normalize the input path
	inputPath := toComplete
	if inputPath == "" {
		inputPath = "/"
	} else if !strings.HasPrefix(inputPath, "/") {
		inputPath = "/" + inputPath
	}

	// Determine parent path and prefix to match
	parentPath := "/"
	prefix := ""

	if inputPath != "/" {
		// Find the last slash to split parent and current component
		lastSlash := strings.LastIndex(inputPath, "/")
		if lastSlash == 0 {
			// Path like "/Doc" - parent is root
			parentPath = "/"
			prefix = strings.ToLower(inputPath[1:])
		} else {
			// Path like "/Folder/Doc" - parent is "/Folder"
			parentPath = inputPath[:lastSlash]
			prefix = strings.ToLower(inputPath[lastSlash+1:])
		}
	}

	// Resolve parent path to storage path
	storagePath, err := resolvePathForCompletion(client, notebook, parentPath)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// List documents in the parent path
	docs, err := client.ListDocsByPath(notebook.ID, storagePath)
	if err != nil {
		// Parent might be a leaf node
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, doc := range docs {
		title := doc.Title()
		titleLower := strings.ToLower(title)

		// Check if document matches the prefix
		if prefix == "" || strings.HasPrefix(titleLower, prefix) {
			var fullPath string
			if parentPath == "/" {
				fullPath = "/" + title
			} else {
				fullPath = parentPath + "/" + title
			}

			// Add trailing slash hint if document has children
			if doc.SubFileCount > 0 {
				completions = append(completions, fullPath+"/")
			} else {
				completions = append(completions, fullPath)
			}
		}
	}

	// Don't add space after completion if it's a directory (has trailing slash)
	return completions, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

// initCompletionClient initializes a minimal client for completion
func initCompletionClient() (*api.Client, *api.Notebook, error) {
	// Read config for completion (viper should already be initialized)
	host := viper.GetString("host")
	if host == "" {
		host = config.DefaultHost
	}
	token := viper.GetString("token")

	client, err := api.NewClient(host, token)
	if err != nil {
		return nil, nil, err
	}

	// Get notebook
	notebookRef := viper.GetString("notebook")
	notebooks, err := client.ListNotebooks()
	if err != nil {
		return nil, nil, err
	}

	var notebook *api.Notebook
	for _, nb := range notebooks {
		if nb.Closed {
			continue
		}
		if notebookRef == "" || nb.Name == notebookRef || nb.ID == notebookRef {
			notebook = nb
			break
		}
	}

	return client, notebook, nil
}

// resolvePathForCompletion resolves a display path to storage path for completion
func resolvePathForCompletion(client *api.Client, notebook *api.Notebook, displayPath string) (string, error) {
	if displayPath == "/" || displayPath == "" {
		return "/", nil
	}

	pathParts := strings.Split(strings.Trim(displayPath, "/"), "/")
	currentPath := "/"

	for _, part := range pathParts {
		docs, err := client.ListDocsByPath(notebook.ID, currentPath)
		if err != nil {
			return "", err
		}

		found := false
		partLower := strings.ToLower(part)
		for _, doc := range docs {
			if strings.ToLower(doc.Title()) == partLower {
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
			return "", nil // Path not found, return empty
		}
	}

	return currentPath, nil
}

// completeNotebook provides completion for notebook names
func completeNotebook(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, _, err := initCompletionClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	notebooks, err := client.ListNotebooks()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	toCompleteLower := strings.ToLower(toComplete)
	for _, nb := range notebooks {
		if nb.Closed {
			continue
		}
		if toComplete == "" || strings.HasPrefix(strings.ToLower(nb.Name), toCompleteLower) {
			completions = append(completions, nb.Name)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
