package client_test

// import (
// 	"context"
// 	"fmt"
// 	"log"

// 	"github.com/notioncodes/client"
// 	"github.com/notioncodes/types"
// )

// // ExampleExportService_ExportDatabase demonstrates how to export a database with all its pages, blocks, and comments
// func ExampleExportService_ExportDatabase() {
// 	config := client.DefaultConfig()
// 	config.APIKey = "your-api-key"

// 	notionClient, err := client.NewClient(config)
// 	if err != nil {
// 		log.Fatalf("Failed to create client: %v", err)
// 	}
// 	defer notionClient.Close()

// 	ctx := context.Background()
// 	databaseID := types.DatabaseID("12345678-1234-5678-9abc-123456789012")

// 	// Create export service
// 	exportService := client.NewExportService(notionClient)

// 	// Export database with full nested data
// 	opts := client.ExportDatabaseOptions{
// 		IncludePages: client.ExportPageOptions{
// 			IncludeBlocks: client.ExportBlockOptions{
// 				IncludeChildren: true,
// 				IncludeComments: client.ExportCommentOptions{IncludeUser: true},
// 			},
// 		},
// 	}

// 	result, err := exportService.ExportDatabase(ctx, databaseID, opts)
// 	if err != nil {
// 		log.Printf("Error exporting database: %v", err)
// 		return
// 	}

// 	fmt.Printf("Database: %s\n", result.Database.Title)
// 	fmt.Printf("Pages: %d\n", len(result.Pages))

// 	for i, page := range result.Pages {
// 		fmt.Printf("  Page %d: %s\n", i+1, page.Page.ID)
// 		fmt.Printf("    Blocks: %d\n", len(page.Blocks))
// 		fmt.Printf("    Comments: %d\n", len(page.Comments))

// 		for j, block := range page.Blocks {
// 			fmt.Printf("      Block %d: %s (children: %d, comments: %d)\n",
// 				j+1, block.Block.ID, len(block.Children), len(block.Comments))

// 			for k, comment := range block.Comments {
// 				fmt.Printf("        Comment %d: %s",
// 					k+1, comment.Comment.GetPlainText())
// 				if comment.User != nil {
// 					fmt.Printf(" by %s", comment.User.Name)
// 				}
// 				fmt.Println()
// 			}
// 		}
// 	}

// 	fmt.Printf("Cached users: %d\n", exportService.GetCachedUserCount())
// }

// // ExampleExportService_ExportPage demonstrates how to export a single page with its blocks and comments
// func ExampleExportService_ExportPage() {
// 	config := client.DefaultConfig()
// 	config.APIKey = "your-api-key"

// 	notionClient, err := client.NewClient(config)
// 	if err != nil {
// 		log.Fatalf("Failed to create client: %v", err)
// 	}
// 	defer notionClient.Close()

// 	ctx := context.Background()
// 	pageID := types.PageID("12345678-1234-5678-9abc-123456789012")

// 	// Create export service
// 	exportService := client.NewExportService(notionClient)

// 	// Export page with blocks and comments, but without user data
// 	opts := client.ExportPageOptions{
// 		IncludeBlocks: client.ExportBlockOptions{
// 			IncludeChildren: true,
// 			IncludeComments: client.ExportCommentOptions{IncludeUser: false},
// 		},
// 	}

// 	result, err := exportService.ExportPage(ctx, pageID, opts)
// 	if err != nil {
// 		log.Printf("Error exporting page: %v", err)
// 		return
// 	}

// 	fmt.Printf("Page: %s\n", result.Page.ID)
// 	fmt.Printf("Blocks: %d\n", len(result.Blocks))
// 	fmt.Printf("Total Comments: %d\n", len(result.Comments))

// 	for i, block := range result.Blocks {
// 		fmt.Printf("  Block %d: %s\n", i+1, block.Block.ID)
// 		fmt.Printf("    Children: %d\n", len(block.Children))
// 		fmt.Printf("    Comments: %d\n", len(block.Comments))
// 	}
// }

// // ExampleExportService_ExportBlock demonstrates how to export a single block with its children and comments
// func ExampleExportService_ExportBlock() {
// 	config := client.DefaultConfig()
// 	config.APIKey = "your-api-key"

// 	notionClient, err := client.NewClient(config)
// 	if err != nil {
// 		log.Fatalf("Failed to create client: %v", err)
// 	}
// 	defer notionClient.Close()

// 	ctx := context.Background()
// 	blockID := types.BlockID("12345678-1234-5678-9abc-123456789012")

// 	// Create export service
// 	exportService := client.NewExportService(notionClient)

// 	// Export block with children but without comments
// 	opts := client.ExportBlockOptions{
// 		IncludeChildren: true,
// 		IncludeComments: client.ExportCommentOptions{IncludeUser: false}, // This will not fetch comments
// 	}

// 	result, err := exportService.ExportBlock(ctx, blockID, opts)
// 	if err != nil {
// 		log.Printf("Error exporting block: %v", err)
// 		return
// 	}

// 	fmt.Printf("Block: %s\n", result.Block.ID)
// 	fmt.Printf("Children: %d\n", len(result.Children))
// 	fmt.Printf("Comments: %d\n", len(result.Comments)) // Should be 0
// }