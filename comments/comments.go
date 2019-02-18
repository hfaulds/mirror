package comments

import (
	"context"
	"io"
	"strings"
	"text/template"

	"github.com/hfaulds/mirror/graphql"
)

type CommentNode struct {
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	URL       string `json:"url"`
	Author    struct {
		Login     string `json:"login"`
		URL       string `json:"url"`
		AvatarURL string `json:"avatarUrl"`
	} `json:"author"`
}

func (i CommentNode) ToMirrorComment() (*CommentNode, error) {
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
	return &CommentNode{
		Body:      body.String(),
		CreatedAt: i.CreatedAt,
		URL:       i.URL,
	}, nil
}

type CommentDiff struct {
	client *graphql.Client
	stdout io.Writer

	mirrorIssueId string
	toCreate      []*CommentNode
}

func NewCommentDiff(client *graphql.Client, stdout io.Writer, mirrorIssueId string, comments []CommentNode) (*CommentDiff, error) {
	diff := &CommentDiff{
		client:        client,
		stdout:        stdout,
		mirrorIssueId: mirrorIssueId,
	}
	for _, comment := range comments {
		mirror, err := comment.ToMirrorComment()
		if err != nil {
			return nil, err
		}
		diff.toCreate = append(diff.toCreate, mirror)
	}
	return diff, nil
}

func (d *CommentDiff) AddNewComments(ctx context.Context) error {
	for _, comment := range d.toCreate {
		d.stdout.Write([]byte("Creating mirror comment\n"))
		vars := map[string]interface{}{
			"subject_id": d.mirrorIssueId,
			"body":       comment.Body,
		}
		var result struct{}
		if err := d.client.Query(ctx, addCommentMutation, vars, &result); err != nil {
			return err
		}
		d.stdout.Write([]byte("Created mirror comment\n"))
	}
	return nil
}
