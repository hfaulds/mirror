package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	graphqlAPIURL = "https://api.github.com/graphql"
)

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse contains the results of a GraphQL query.
type GraphQLResponse struct {
	Data   interface{}    `json:"data"`
	Errors []GraphQLError `json:"errors"`
}

// GraphQLError contains data relating to validation errors in a GraphQL query.
type GraphQLError struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

func (e GraphQLError) Error() string {
	return "error from API: " + e.Message
}

type Client struct {
	httpClient *http.Client
	token      string
}

func NewClient(token string) *Client {
	return &Client{token: token, httpClient: &http.Client{}}
}

func (c *Client) Query(ctx context.Context, query string, variables map[string]interface{}, dest interface{}) error {
	req := &graphQLRequest{Query: query, Variables: variables}
	resp := &GraphQLResponse{Data: dest}
	if err := c.makeRequest(ctx, req, resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("Unexpected error: %#v", resp.Errors)
	}
	return nil
}

func (c *Client) makeRequest(ctx context.Context, request *graphQLRequest, response *GraphQLResponse) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", graphqlAPIURL, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(response); err != nil {
		return err
	}

	return nil
}
