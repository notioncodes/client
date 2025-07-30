package client_test

// import (
// 	"context"
// 	"fmt"
// 	"log"

// 	"github.com/notioncodes/client"
// 	"github.com/notioncodes/types"
// )

// // ExamplePageNamespace_Get demonstrates how to retrieve a page with blocks and comments
// func ExamplePageNamespace_Get() {
// 	config := client.DefaultConfig()
// 	config.APIKey = "your-api-key"

// 	notionClient, err := client.NewClient(config)
// 	if err != nil {
// 		log.Fatalf("Failed to create client: %v", err)
// 	}
// 	defer notionClient.Close()

// 	ctx := context.Background()
// 	pageID := types.PageID("12345678-1234-5678-9abc-123456789012")

// 	// Example 1: Get page with blocks and comments
// 	opts := client.GetPageOptions{
// 		IncludeBlocks:   true,
// 		IncludeComments: true,
// 		IncludeChildren: true, // Recursive block retrieval
// 	}

// 	result := notionClient.Registry.Pages().Get(ctx, pageID, opts)
// 	if result.IsError() {
// 		log.Printf("Error: %v", result.Error)
// 		return
// 	}

// 	pageData := result.Data
// 	fmt.Printf("Page Title: %s\n", pageData.Page.Title)
// 	fmt.Printf("Number of blocks: %d\n", len(pageData.Blocks))
// 	fmt.Printf("Number of comments: %d\n", len(pageData.Comments))

// 	// Example 2: Get simple page without additional data
// 	simpleResult := notionClient.Registry.Pages().GetSimple(ctx, pageID)
// 	if simpleResult.IsError() {
// 		log.Printf("Error: %v", simpleResult.Error)
// 		return
// 	}

// 	fmt.Printf("Simple page title: %s\n", simpleResult.Data.Title)
// }

// // ExamplePageNamespace_GetMany demonstrates how to retrieve multiple pages with blocks
// func ExamplePageNamespace_GetMany() {
// 	config := client.DefaultConfig()
// 	config.APIKey = "your-api-key"

// 	notionClient, err := client.NewClient(config)
// 	if err != nil {
// 		log.Fatalf("Failed to create client: %v", err)
// 	}
// 	defer notionClient.Close()

// 	ctx := context.Background()
// 	pageIDs := []types.PageID{
// 		types.PageID("12345678-1234-5678-9abc-123456789012"),
// 		types.PageID("87654321-4321-8765-cba9-210987654321"),
// 	}

// 	// Get multiple pages with blocks
// 	opts := client.GetPageOptions{
// 		IncludeBlocks:   true,
// 		IncludeChildren: true,
// 	}

// 	results := notionClient.Registry.Pages().GetMany(ctx, pageIDs, opts)
// 	for result := range results {
// 		if result.IsError() {
// 			log.Printf("Error: %v", result.Error)
// 			continue
// 		}

// 		pageData := result.Data
// 		fmt.Printf("Page: %s, Blocks: %d\n",
// 			pageData.Page.Title,
// 			len(pageData.Blocks))
// 	}
// }
