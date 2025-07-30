package client

import (
	"context"
	"fmt"
	"testing"

	"github.com/notioncodes/test"
	"github.com/notioncodes/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type GetPageSuite struct {
	suite.Suite
	client *Client
	ids    []types.PageID
	opts   GetPageOptions
}

func (s *GetPageSuite) SetupSuite() {
	s.ids = []types.PageID{
		"23fd7342e571819596ccfb5fbb9144f7",
		// "23fd7342e571814ead4df15eaf81d23f",
	}

	s.opts = GetPageOptions{
		IncludeBlocks:      true,
		IncludeComments:    true,
		IncludeAttachments: true,
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

func TestGetPageSuite(t *testing.T) {
	suite.Run(t, new(GetPageSuite))
}

func (s *GetPageSuite) TestGetPage() {
	result := s.client.Pages().Get(context.Background(), s.ids[0], s.opts)
	s.NoError(result.Error)
	s.NotNil(result.Data)
}

func (s *GetPageSuite) TestGetPageMany() {
	results := s.client.Pages().GetMany(context.Background(), s.ids, s.opts)
	for result := range results {
		assert.NoError(s.T(), result.Error)
		assert.Greater(s.T(), len(result.Data.Blocks), 0, fmt.Sprintf("%s: expected blocks > 0", result.Data.Page.ID))
		assert.Greater(s.T(), len(result.Data.Comments), 0, fmt.Sprintf("%s: expected comments > 0", result.Data.Page.ID))
		assert.Greater(s.T(), len(result.Data.Attachments), 0, fmt.Sprintf("%s: expected attachments > 0", result.Data.Page.ID))
	}
}
