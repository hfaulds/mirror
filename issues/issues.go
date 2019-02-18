package issues

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/hfaulds/mirror/graphql"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type issueNode struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	URL       string `json:"url"`
	Author    struct {
		Login     string `json:"login"`
		URL       string `json:"url"`
		AvatarURL string `json:"avatarUrl"`
	} `json:"author"`
	Comments struct {
		Nodes []struct {
			Body string `json:"body"`
		} `json:"nodes"`
	} `json:"comments"`
}

func (i issueNode) ToMirrorIssue() (*issueNode, error) {
	mirrorTemplate, err := template.New("Mirror Issue Template").Parse(`
<img class="avatar rounded-1" height="44" width="44" align="left" alt="@hayden-bot" src="{{.Author.AvatarURL}}"></img>

## [{{.Author.Login}}]({{.Author.URL}}) at [{{.CreatedAt}}]({{.URL}}):

> {{.Body}}`)
	if err != nil {
		return nil, err
	}
	var body strings.Builder
	if err = mirrorTemplate.Execute(&body, i); err != nil {
		return nil, err
	}
	return &issueNode{
		Number:   i.Number,
		Title:    fmt.Sprintf("Mirror #%d: %s", i.Number, i.Title),
		Body:     body.String(),
		Author:   i.Author,
		Comments: i.Comments,
	}, nil
}

type queryResponseData struct {
	Repository struct {
		ID     string `json:"id"`
		Issues struct {
			Nodes []issueNode `json:"nodes"`
		} `json:"issues"`
	} `json:"repository"`
}

type issueDiffer struct {
	fromClient *graphql.Client
	toClient   *graphql.Client
	stdout     io.Writer
	from       string
	to         string
}

type issueDiff struct {
	client *graphql.Client
	stdout io.Writer

	mirrorRepoId string
	toCreate     []*issueNode
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
	fromClient := graphql.NewClient(viper.GetString("github_token"))
	toClient := graphql.NewClient(viper.GetString("to_token"))

	differ := NewDiffer(fromClient, toClient, stdout, viper.GetString("from"), viper.GetString("to"))
	diff, err := differ.Diff(ctx)
	if err != nil {
		return err
	}
	return diff.OpenNewIssues(ctx)
	//closeOldIssues
	//updateChangedIssues
}

func NewDiffer(fromClient, toClient *graphql.Client, stdout io.Writer, from, to string) issueDiffer {
	return issueDiffer{
		fromClient: fromClient,
		toClient:   toClient,
		stdout:     stdout,
		from:       from,
		to:         to,
	}
}

func (d issueDiffer) Diff(ctx context.Context) (*issueDiff, error) {
	fromOwner, fromName := parseNWO(d.from)
	fromResp, err := fetchIssues(ctx, d.fromClient, fromOwner, fromName)
	if err != nil {
		return nil, err
	}

	toOwner, toName := parseNWO(d.to)
	toResp, err := fetchIssues(ctx, d.toClient, toOwner, toName)
	if err != nil {
		return nil, err
	}

	diff := &issueDiff{
		client:       d.toClient,
		stdout:       d.stdout,
		mirrorRepoId: toResp.Repository.ID,
	}
	mirrorIssues := toResp.Repository.Issues.Nodes
	mirrorIssuesMap := make(map[string]issueNode, len(mirrorIssues))
	for _, issue := range mirrorIssues {
		mirrorIssuesMap[issue.Title] = issue
	}

	for _, issue := range fromResp.Repository.Issues.Nodes {
		mirror, err := issue.ToMirrorIssue()
		if err != nil {
			return nil, err
		}
		if _, ok := mirrorIssuesMap[mirror.Title]; !ok {
			diff.toCreate = append(diff.toCreate, mirror)
		}
	}

	return diff, nil
}

func fetchIssues(ctx context.Context, client *graphql.Client, owner, name string) (*queryResponseData, error) {
	vars := map[string]interface{}{
		"owner": owner,
		"name":  name,
	}
	var data queryResponseData
	err := client.Query(ctx, getIssueQuery, vars, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (d issueDiff) OpenNewIssues(ctx context.Context) error {
	for _, issue := range d.toCreate {
		d.stdout.Write([]byte(fmt.Sprintf("Created mirror issue for #%d", issue.Number)))
		vars := map[string]interface{}{
			"repository_id": d.mirrorRepoId,
			"title":         issue.Title,
			"body":          issue.Body,
		}
		var mirror issueNode
		if err := d.client.Query(ctx, createIssueMutation, vars, &mirror); err != nil {
			return err
		}
		d.stdout.Write([]byte(fmt.Sprintf("Created mirror issue #%d -> #%d", issue.Number, mirror.Number)))
	}
	return nil
}

func handleErrors(f func(w io.Writer) error) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		err := f(os.Stdout)
		if err != nil {
			fmt.Printf("Error syncing issues: %s\n", err)
			os.Exit(-1)
		}
	}
}

func parseNWO(raw string) (string, string) {
	split := strings.SplitN(raw, "/", 2)
	if len(split) != 2 {
		return "", ""
	}
	return split[0], split[1]
}
