package issues

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hfaulds/mirror/graphql"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const getIssuesQuery = `
query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
		id
    issues(first: 100) {
      nodes {
				number
        title
        body
        author {
          login
          url
        }
        comments(first: 100) {
          nodes {
            body
          }
        }
      }
    }
  }
}
`

const createIssuesMutation = `
mutation($repository_id: ID!, $title: String!, $body: String!) {
	createIssue(input: {
		repositoryId: $repository_id,
		title: $title,
		body: $body,
	}) {
		issue {
			number
		}
	}
}
`

type issueNode struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	Author struct {
		Login string `json:"login"`
		URL   string `url:login`
	} `json:"author"`
	Comments struct {
		Nodes []struct {
			Body string `json:"body"`
		} `json:"nodes"`
	} `json:"comments"`
}

func (i issueNode) WithMirrorTitle() issueNode {
	return issueNode{
		Number:   i.Number,
		Title:    fmt.Sprintf("Mirror #%d: %s", i.Number, i.Title),
		Body:     i.Body,
		Author:   i.Author,
		Comments: i.Comments,
	}
}

type queryResponseData struct {
	Repository struct {
		ID     string `json:"id"`
		Issues struct {
			Nodes []issueNode `json:"nodes"`
		} `json:"issues"`
	} `json:"repository"`
}

type issueSync struct {
	client *graphql.Client
	stdout io.Writer

	mirrorRepoId string
	toCreate     []issueNode
}

func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "issues",
		Short: "Sync issues",
		Run:   handleErrors(syncIssues),
	}
}

func syncIssues(stdout io.Writer) error {
	ctx := context.Background()
	client := createClient()

	fromOwner, fromName := parseNWO(viper.GetString("from"))
	fromResp, err := fetchIssues(ctx, client, fromOwner, fromName)
	if err != nil {
		return err
	}

	toOwner, toName := parseNWO(viper.GetString("to"))
	toResp, err := fetchIssues(ctx, client, toOwner, toName)
	if err != nil {
		return err
	}

	syncer := NewIssueSync(client, stdout, fromResp, toResp)
	return syncer.openNewIssues(ctx)
	//closeOldIssues
	//updateChangedIssues
}

func parseNWO(raw string) (string, string) {
	split := strings.SplitN(raw, "/", 2)
	return split[0], split[1]
}

func fetchIssues(ctx context.Context, client *graphql.Client, owner, name string) (*queryResponseData, error) {
	vars := map[string]interface{}{
		"owner": owner,
		"name":  name,
	}
	var data queryResponseData
	err := client.Query(ctx, getIssuesQuery, vars, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func NewIssueSync(client *graphql.Client, stdout io.Writer, fromResp, toResp *queryResponseData) issueSync {
	syncer := issueSync{
		client:       client,
		stdout:       stdout,
		mirrorRepoId: toResp.Repository.ID,
	}

	mirrorIssues := toResp.Repository.Issues.Nodes
	mirrorIssuesMap := make(map[string]issueNode, len(mirrorIssues))
	for _, issue := range mirrorIssues {
		mirrorIssuesMap[issue.Title] = issue
	}

	for _, issue := range fromResp.Repository.Issues.Nodes {
		mirror := issue.WithMirrorTitle()
		if _, ok := mirrorIssuesMap[mirror.Title]; !ok {
			syncer.toCreate = append(syncer.toCreate, mirror)
		}
	}

	return syncer
}

func (s *issueSync) openNewIssues(ctx context.Context) error {
	for _, issue := range s.toCreate {
		s.stdout.Write([]byte(fmt.Sprintf("Created mirror issue for #%d", issue.Number)))
		vars := map[string]interface{}{
			"repository_id": s.mirrorRepoId,
			"title":         issue.Title,
			"body":          issue.Body,
		}
		var mirror issueNode
		if err := s.client.Query(ctx, createIssuesMutation, vars, &mirror); err != nil {
			return err
		}
		s.stdout.Write([]byte(fmt.Sprintf("Created mirror issue #%d -> #%d", issue.Number, mirror.Number)))
	}
	return nil
}

func createClient() *graphql.Client {
	return graphql.NewClient(viper.GetString("token"))
}

func handleErrors(f func(w io.Writer) error) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		err := f(os.Stdout)
		if err != nil {
			fmt.Printf("Error syncing issues: %s", err)
			os.Exit(-1)
		}
	}
}
