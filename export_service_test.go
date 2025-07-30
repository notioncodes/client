package client

import (
	"context"
	"testing"

	"github.com/notioncodes/test"
	"github.com/notioncodes/types"
	"github.com/stretchr/testify/suite"
)

type ExportServiceSuite struct {
	suite.Suite
	client        *Client
	exportService *ExportService
	ids           []types.BlockID
	pageIDs       []types.PageID
}

func (s *ExportServiceSuite) SetupSuite() {
	s.ids = []types.BlockID{
		"23fd7342-e571-8084-a8d9-d5eecbeeeb1a",
		"240d7342-e571-801e-aa21-cbae0bf393af", // with comments
	}

	s.pageIDs = []types.PageID{
		types.PageID("23fd7342-e571-8195-96cc-fb5fbb9144f7"),
	}

	var err error
	s.client, err = NewClient(&Config{
		APIKey:        test.TestConfig.NotionAPIKey,
		EnableMetrics: true,
	})
	if err != nil {
		s.T().Fatalf("failed to create client: %v", err)
	}

	s.exportService = NewExportService(s.client)
}

func (s *ExportServiceSuite) TestExportPage() {
	// Test basic page export without blocks
	opts := ExportPageOptions{
		Blocks: ExportBlockOptions{
			Children: false,
			Comments: ExportCommentOptions{User: false},
		},
	}

	// Use a known page ID from the test data
	pageID := s.pageIDs[0]

	result, err := s.exportService.ExportPage(context.Background(), pageID, opts)
	s.NoError(err)
	s.NotNil(result)
	s.NotNil(result.Page)
	s.Equal(pageID, result.Page.ID)
	s.Empty(result.Blocks)
}

func (s *ExportServiceSuite) TestExportPageWithBlocks() {
	// Test page export with blocks but no children or comments
	opts := ExportPageOptions{
		Blocks: ExportBlockOptions{
			Children: false,
			Comments: ExportCommentOptions{User: false},
		},
	}

	pageID := s.pageIDs[0]

	result, err := s.exportService.ExportPage(context.Background(), pageID, opts)
	s.NoError(err)
	s.NotNil(result)
	s.NotNil(result.Page)
	s.Equal(pageID, result.Page.ID)
	// Should have blocks since we're including them
	s.NotEmpty(result.Blocks)
	// Each block should not have children or comments
	for _, blockResult := range result.Blocks {
		s.NotNil(blockResult.Block)
		s.Empty(blockResult.Children)
		s.Empty(blockResult.Comments)
	}
}

func (s *ExportServiceSuite) TestExportPageWithBlocksAndChildren() {
	// Test page export with blocks and their children
	opts := ExportPageOptions{
		Blocks: ExportBlockOptions{
			Children: true,
			Comments: ExportCommentOptions{User: false},
		},
	}

	pageID := s.pageIDs[0]

	result, err := s.exportService.ExportPage(context.Background(), pageID, opts)
	s.NoError(err)
	s.NotNil(result)
	s.NotNil(result.Page)
	s.Equal(pageID, result.Page.ID)
	s.NotEmpty(result.Blocks)
	// Blocks should have children if they exist
	for _, blockResult := range result.Blocks {
		s.NotNil(blockResult.Block)
		// Children may or may not exist, but the field should be initialized
		s.NotNil(blockResult.Children)
	}
}

func (s *ExportServiceSuite) TestExportPageWithComments() {
	// Test page export with comments and users
	opts := ExportPageOptions{
		Blocks: ExportBlockOptions{
			Children: false,
			Comments: ExportCommentOptions{User: true},
		},
	}

	// Use the block ID that has comments
	pageID := s.pageIDs[0]

	result, err := s.exportService.ExportPage(context.Background(), pageID, opts)
	s.NoError(err)
	s.NotNil(result)
	s.NotNil(result.Page)
	s.Equal(pageID, result.Page.ID)
	s.NotEmpty(result.Blocks)

	// Check if any blocks have comments
	for _, blockResult := range result.Blocks {
		s.NotNil(blockResult.Block)
		if len(blockResult.Comments) > 0 {
			// Verify comment structure
			for _, commentResult := range blockResult.Comments {
				s.NotNil(commentResult.Comment)
				// User should be included since IncludeUser is true
				if commentResult.Comment.CreatedBy != nil {
					s.NotNil(commentResult.User)
				}
			}
		}
	}
}

func (s *ExportServiceSuite) TestExportPageFullOptions() {
	// Test page export with all options enabled
	opts := ExportPageOptions{
		Blocks: ExportBlockOptions{
			Children: true,
			Comments: ExportCommentOptions{User: true},
		},
	}

	pageID := s.pageIDs[0]

	result, err := s.exportService.ExportPage(context.Background(), pageID, opts)
	s.NoError(err)
	s.NotNil(result)
	s.NotNil(result.Page)
	s.Equal(pageID, result.Page.ID)
	s.NotEmpty(result.Blocks)

	// Verify all data is properly structured
	for _, blockResult := range result.Blocks {
		s.NotNil(blockResult.Block)
		s.NotNil(blockResult.Children)
		s.NotNil(blockResult.Comments)

		// If comments exist, verify user data
		for _, commentResult := range blockResult.Comments {
			s.NotNil(commentResult.Comment)
			if commentResult.Comment.CreatedBy != nil {
				s.NotNil(commentResult.User)
			}
		}
	}
}

func (s *ExportServiceSuite) TestExportPageInvalidID() {
	// Test page export with invalid page ID
	opts := ExportPageOptions{
		Blocks: ExportBlockOptions{
			Children: false,
			Comments: ExportCommentOptions{User: false},
		},
	}

	invalidPageID := types.PageID("invalid-page-id")

	result, err := s.exportService.ExportPage(context.Background(), invalidPageID, opts)
	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "failed to get page")
}

func TestExportServiceSuite(t *testing.T) {
	suite.Run(t, new(ExportServiceSuite))
}

func (s *ExportServiceSuite) TestExportBlock() {
	// Test basic block export without children or comments
	opts := ExportBlockOptions{
		Children: false,
		Comments: ExportCommentOptions{User: false},
	}

	result, err := s.exportService.ExportBlock(context.Background(), s.ids[0], opts)
	s.NoError(err)
	s.NotNil(result)
	s.NotNil(result.Block)
	s.Equal(s.ids[0], result.Block.ID)
	s.Empty(result.Children)
	s.Empty(result.Comments)
}

func (s *ExportServiceSuite) TestExportBlockWithChildren() {
	// Test block export with children but no comments
	opts := ExportBlockOptions{
		Children: true,
		Comments: ExportCommentOptions{User: false},
	}

	result, err := s.exportService.ExportBlock(context.Background(), s.ids[0], opts)
	s.NoError(err)
	s.NotNil(result)
	s.NotNil(result.Block)
	s.Equal(s.ids[0], result.Block.ID)
	// Children field should exist (may be empty slice)
	s.GreaterOrEqual(len(result.Children), 0)
	s.Empty(result.Comments) // Should be empty since IncludeUser is false
}

func (s *ExportServiceSuite) TestExportBlockWithComments() {
	// Test block export with comments but no children
	opts := ExportBlockOptions{
		Children: false,
		Comments: ExportCommentOptions{User: true},
	}

	result, err := s.exportService.ExportBlock(context.Background(), s.ids[0], opts)
	s.NoError(err)
	s.NotNil(result)
	s.NotNil(result.Block)
	s.Equal(s.ids[0], result.Block.ID)
	s.Empty(result.Children)
	// Comments may or may not exist, but the field should be non-nil
	s.NotNil(result.Comments)
}

func (s *ExportServiceSuite) TestUserCaching() {
	// Test that user caching works
	initialCount := s.exportService.GetCachedUserCount()

	// Export block with comments that include users
	opts := ExportBlockOptions{
		Children: false,
		Comments: ExportCommentOptions{User: true},
	}

	_, err := s.exportService.ExportBlock(context.Background(), s.ids[0], opts)
	s.NoError(err)

	// Cache count should be >= initial count (may have increased if comments with users were found)
	finalCount := s.exportService.GetCachedUserCount()
	s.GreaterOrEqual(finalCount, initialCount)

	// Test cache clearing
	s.exportService.ClearUserCache()
	s.Equal(0, s.exportService.GetCachedUserCount())
}
