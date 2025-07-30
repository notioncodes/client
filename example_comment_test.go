package client_test

// import (
// 	"context"
// 	"fmt"
// 	"log"

// 	"github.com/notioncodes/client"
// 	"github.com/notioncodes/types"
// )

// // ExampleCommentNamespace_List demonstrates how to retrieve comments from a page
// func ExampleCommentNamespace_List() {
// 	config := client.DefaultConfig()
// 	config.APIKey = "your-api-key"

// 	notionClient, err := client.NewClient(config)
// 	if err != nil {
// 		log.Fatalf("Failed to create client: %v", err)
// 	}
// 	defer notionClient.Close()

// 	ctx := context.Background()
// 	pageID := types.PageID("12345678-1234-5678-9abc-123456789012")

// 	// List comments from a page
// 	results := notionClient.Registry.Comments().ListByPage(ctx, pageID)
// 	for result := range results {
// 		if result.IsError() {
// 			log.Printf("Error: %v", result.Error)
// 			continue
// 		}

// 		comment := result.Data
// 		fmt.Printf("Comment ID: %s\n", comment.ID)
// 		fmt.Printf("Comment Text: %s\n", comment.GetPlainText())
// 		fmt.Printf("Discussion ID: %s\n", comment.DiscussionID)

// 		if comment.CreatedBy != nil {
// 			fmt.Printf("Created by: %s\n", comment.CreatedBy.Name)
// 		}
// 	}
// }

// // ExampleCommentNamespace_Create demonstrates how to create a comment on a page
// func ExampleCommentNamespace_Create() {
// 	config := client.DefaultConfig()
// 	config.APIKey = "your-api-key"

// 	notionClient, err := client.NewClient(config)
// 	if err != nil {
// 		log.Fatalf("Failed to create client: %v", err)
// 	}
// 	defer notionClient.Close()

// 	ctx := context.Background()
// 	pageID := types.PageID("12345678-1234-5678-9abc-123456789012")

// 	// Create rich text for the comment
// 	richText := []types.RichText{
// 		*types.NewTextRichText("This is a great page! ", nil),
// 		*types.NewTextRichText("Keep up the good work.", &types.Annotations{
// 			Bold: true,
// 		}),
// 	}

// 	// Create comment on page
// 	result := notionClient.Registry.Comments().CreateOnPage(ctx, pageID, richText)
// 	if result.IsError() {
// 		log.Printf("Error creating comment: %v", result.Error)
// 		return
// 	}

// 	comment := result.Data
// 	fmt.Printf("Created comment ID: %s\n", comment.ID)
// 	fmt.Printf("Comment text: %s\n", comment.GetPlainText())
// }

// // ExampleCommentNamespace_CreateInDiscussion demonstrates replying to a comment thread
// func ExampleCommentNamespace_CreateInDiscussion() {
// 	config := client.DefaultConfig()
// 	config.APIKey = "your-api-key"

// 	notionClient, err := client.NewClient(config)
// 	if err != nil {
// 		log.Fatalf("Failed to create client: %v", err)
// 	}
// 	defer notionClient.Close()

// 	ctx := context.Background()
// 	discussionID := types.DiscussionID("87654321-4321-8765-cba9-210987654321")

// 	// Create reply in discussion thread
// 	richText := []types.RichText{
// 		*types.NewTextRichText("I completely agree with this point!", nil),
// 	}

// 	result := notionClient.Registry.Comments().CreateInDiscussion(ctx, discussionID, richText)
// 	if result.IsError() {
// 		log.Printf("Error creating reply: %v", result.Error)
// 		return
// 	}

// 	comment := result.Data
// 	fmt.Printf("Created reply ID: %s\n", comment.ID)
// 	fmt.Printf("Reply text: %s\n", comment.GetPlainText())
// }

// // ExampleCommentNamespace_GetAll demonstrates getting all comments as a slice
// func ExampleCommentNamespace_GetAll() {
// 	config := client.DefaultConfig()
// 	config.APIKey = "your-api-key"

// 	notionClient, err := client.NewClient(config)
// 	if err != nil {
// 		log.Fatalf("Failed to create client: %v", err)
// 	}
// 	defer notionClient.Close()

// 	ctx := context.Background()
// 	pageID := types.PageID("12345678-1234-5678-9abc-123456789012")

// 	// Get all comments for a page as a slice
// 	comments, err := notionClient.Registry.Comments().GetAllByPage(ctx, pageID)
// 	if err != nil {
// 		log.Printf("Error: %v", err)
// 		return
// 	}

// 	fmt.Printf("Found %d comments on the page\n", len(comments))

// 	for i, comment := range comments {
// 		fmt.Printf("Comment %d: %s\n", i+1, comment.GetPlainText())

// 		// Check if it's a page or block comment
// 		if comment.IsPageComment() {
// 			fmt.Printf("  -> This is a page comment\n")
// 		} else if comment.IsBlockComment() {
// 			fmt.Printf("  -> This is a block comment\n")
// 		}
// 	}
// }

// // ExampleBlockNamespace_GetWithComments demonstrates getting blocks with their comments
// func ExampleBlockNamespace_GetWithComments() {
// 	config := client.DefaultConfig()
// 	config.APIKey = "your-api-key"

// 	notionClient, err := client.NewClient(config)
// 	if err != nil {
// 		log.Fatalf("Failed to create client: %v", err)
// 	}
// 	defer notionClient.Close()

// 	ctx := context.Background()
// 	blockID := types.BlockID("12345678-1234-5678-9abc-123456789012")

// 	// Get block with comments
// 	opts := client.GetBlockOptions{
// 		IncludeComments: true,
// 	}

// 	result := notionClient.Registry.Blocks().Get(ctx, blockID, opts)
// 	if result.IsError() {
// 		log.Printf("Error: %v", result.Error)
// 		return
// 	}

// 	blockData := result.Data
// 	fmt.Printf("Block: %s\n", blockData.Block.ID)
// 	fmt.Printf("Comments: %d\n", len(blockData.Comments))

// 	// Process comments
// 	for _, comment := range blockData.Comments {
// 		fmt.Printf("Comment: %s (Discussion: %s)\n",
// 			comment.GetPlainText(),
// 			comment.DiscussionID)
// 	}
// }

// // ExampleBlockNamespace_GetChildrenWithComments demonstrates getting page blocks with comments
// func ExampleBlockNamespace_GetChildrenWithComments() {
// 	config := client.DefaultConfig()
// 	config.APIKey = "your-api-key"

// 	notionClient, err := client.NewClient(config)
// 	if err != nil {
// 		log.Fatalf("Failed to create client: %v", err)
// 	}
// 	defer notionClient.Close()

// 	ctx := context.Background()
// 	pageID := types.PageID("12345678-1234-5678-9abc-123456789012")

// 	// Get page blocks with comments
// 	opts := client.GetBlockOptions{
// 		IncludeComments: true,
// 	}

// 	results := notionClient.Registry.Blocks().GetChildrenWithOptions(ctx, types.BlockID(pageID.String()), opts)
// 	for result := range results {
// 		if result.IsError() {
// 			log.Printf("Error: %v", result.Error)
// 			continue
// 		}

// 		blockData := result.Data
// 		fmt.Printf("Block: %s, Comments: %d\n", blockData.Block.ID, len(blockData.Comments))

// 		// Process comments for this block
// 		for _, comment := range blockData.Comments {
// 			fmt.Printf("  Comment: %s (Discussion: %s)\n",
// 				comment.GetPlainText(),
// 				comment.DiscussionID)
// 		}
// 	}
// }
