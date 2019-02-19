package issues

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hfaulds/mirror/comments"
	"github.com/hfaulds/mirror/graphql"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type issueNode struct {
	ID        string `json:"id"`
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
		Nodes []comments.CommentNode `json:"nodes"`
	} `json:"comments"`
}

func (i issueNode) originalComment() comments.CommentNode {
	return comments.CommentNode{
		Body:      i.Body,
		CreatedAt: i.CreatedAt,
		URL:       i.URL,
		Author:    i.Author,
	}
}

func (i issueNode) ToMirrorIssue() (*issueNode, error) {
	originalComment, err := i.originalComment().ToMirrorComment()
	if err != nil {
		return nil, err
	}
	return &issueNode{
		Number:    i.Number,
		Title:     fmt.Sprintf("Mirror #%d: %s", i.Number, i.Title),
		Body:      originalComment.Body,
		CreatedAt: i.CreatedAt,
		URL:       i.URL,
		Author:    i.Author,
		Comments:  i.Comments,
	}, nil
}

func (i issueNode) CommentNodesByAuthor(login string) []comments.CommentNode {
	var comments []comments.CommentNode
	for _, comment := range i.Comments.Nodes {
		if comment.Author.Login == login {
			comments = append(comments, comment)
		}
	}
	return comments
}

type queryResponseData struct {
	Repository struct {
		ID     string `json:"id"`
		Issues struct {
			Nodes []issueNode `json:"nodes"`
		} `json:"issues"`
	} `json:"repository"`
}

type issueDiff struct {
	client *graphql.Client
	stdout io.Writer

	mirrorRepoId  string
	toCreate      []*issueNode
	commentsToAdd []*comments.CommentDiff
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
	if err := diff.OpenNewIssues(ctx); err != nil {
		return err
	}
	return diff.UpdateIssuesWithNewComments(ctx)
	//closeOldIssues
	//updateChangedIssues
}

func (d issueDiff) OpenNewIssues(ctx context.Context) error {
	for _, issue := range d.toCreate {
		d.stdout.Write([]byte(fmt.Sprintf("Creating mirror issue for #%d\n", issue.Number)))
		vars := map[string]interface{}{
			"repository_id": d.mirrorRepoId,
			"title":         issue.Title,
			"body":          issue.Body,
		}

		var result struct {
			CreateIssue struct {
				Issue struct {
					ID     string `json:"id"`
					Number int    `json:"number"`
				} `json:"issue"`
			} `json:"createIssue"`
		}
		err := d.client.Query(ctx, createIssueMutation, vars, &result)
		if err != nil {
			return err
		}
		newIssue := result.CreateIssue.Issue
		d.stdout.Write([]byte(fmt.Sprintf("Created mirror issue #%d -> #%d\n", issue.Number, newIssue.Number)))
		commentDiff, err := comments.NewCommentDiff(
			d.client,
			d.stdout,
			newIssue.ID,
			issue.Comments.Nodes,
		)
		if err != nil {
			return err
		}
		if err := commentDiff.AddNewComments(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (d issueDiff) UpdateIssuesWithNewComments(ctx context.Context) error {
	for _, commentDiff := range d.commentsToAdd {
		if err := commentDiff.AddNewComments(ctx); err != nil {
			return err
		}
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
