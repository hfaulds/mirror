package issues

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hfaulds/mirror/graphql"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const getIssuesQuery = `
query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    issues(first: 9) {
      nodes {
        title
        bodyText
        author {
          login
          url
        }
        comments(first: 10) {
          nodes {
            bodyText
          }
        }
      }
    }
  }
}
`

type issueNode struct {
	Title    string `json:"title"`
	BodyText string `json:"body_text"`
	Author   struct {
		Login string `json:"login"`
		URL   string `url:login`
	} `json:"author"`
	Comments struct {
		Nodes []struct {
			BodyText string `json:"bodyText"`
		} `json:"nodes"`
	} `json:"comments"`
}

type responseData struct {
	Repository struct {
		Issues struct {
			Nodes []issueNode `json:"nodes"`
		} `json:"issues"`
	} `json:"repository"`
}

func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "issues",
		Short: "Sync issues",
		Run:   handleErrors(syncIssues),
	}
}

func syncIssues() error {
	ctx := context.Background()
	client := createClient()

	fromOwner, fromName := parseNWO(viper.GetString("from"))
	fromIssues, err := fetchIssues(ctx, client, fromOwner, fromName)
	if err != nil {
		return err
	}

	toOwner, toName := parseNWO(viper.GetString("to"))
	toIssues, err := fetchIssues(ctx, client, toOwner, toName)
	if err != nil {
		return err
	}

	fmt.Println(fromIssues)
	fmt.Println(toIssues)

	return nil
}

func parseNWO(raw string) (string, string) {
	split := strings.SplitN(raw, "/", 2)
	return split[0], split[1]
}

func fetchIssues(ctx context.Context, client *graphql.Client, owner, name string) ([]issueNode, error) {
	vars := map[string]interface{}{"owner": owner, "name": name}
	var data responseData
	err := client.Query(ctx, getIssuesQuery, vars, &data)
	if err != nil {
		return nil, err
	}
	return data.Repository.Issues.Nodes, nil
}

func createClient() *graphql.Client {
	return graphql.NewClient(viper.GetString("token"))
}

func handleErrors(f func() error) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		err := f()
		if err != nil {
			os.Exit(-1)
			fmt.Errorf("Error syncing issues: %s", err)
		}
	}
}
