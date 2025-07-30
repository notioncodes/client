package client

import (
	"context"
	"testing"

	"github.com/notioncodes/test"
	"github.com/notioncodes/types"
	"github.com/stretchr/testify/suite"
)

type GetCommentSuite struct {
	suite.Suite
	client *Client
	ids    []types.BlockID
	opts   GetPageOptions
}

func (s *GetCommentSuite) SetupSuite() {
	s.ids = []types.BlockID{
		"240d7342-e571-801e-aa21-cbae0bf393af",
	}

	s.opts = GetPageOptions{
		IncludeBlocks:      true,
		IncludeAttachments: true,
		IncludeChildren:    true,
	}

	var err error
	s.client, err = NewClient(&Config{
		APIKey:        test.TestConfig.NotionAPIKey,
		EnableMetrics: true,
	})
	if err != nil {
		s.T().Fatalf("failed to create client: %v", err)
	}
}

func TestGetCommentSuite(t *testing.T) {
	suite.Run(t, new(GetCommentSuite))
}

func (s *GetCommentSuite) TestGetComment() {
	// Test basic block retrieval without comments
	result := s.client.Blocks().GetWithOptions(context.Background(), s.ids[0], GetBlockOptions{
		IncludeComments: false,
	})
	s.NoError(result.Error)
	s.NotNil(result.Data)
	s.NotNil(result.Data.Block)

	// Test that comments field is nil when not requested
	s.Nil(result.Data.Comments)
}

func (s *GetCommentSuite) TestGetCommentWithComments() {
	// Test block retrieval with comments (may fail if block has no comments or API issues)
	result := s.client.Comments().List(context.Background(), &ListCommentsOptions{
		BlockID: &s.ids[0],
	})
	for result := range result {
		s.NoError(result.Error)
		s.NotNil(result.Data)
	}
}
