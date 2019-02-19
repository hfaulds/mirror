package issues

import (
	"context"
	"io"

	"github.com/hfaulds/mirror/comments"
	"github.com/hfaulds/mirror/graphql"
)

type issueDiffer struct {
	fromClient *graphql.Client
	toClient   *graphql.Client
	stdout     io.Writer
	from       string
	to         string
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

	botLogin, err := getViewerLogin(ctx, d.toClient)
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
		expectedMirror, err := issue.ToMirrorIssue()
		if err != nil {
			return nil, err
		}
		if actualMirror, ok := mirrorIssuesMap[expectedMirror.Title]; ok {
			commentDiff, hasDifference, err := d.diffIssueComments(botLogin, actualMirror, *expectedMirror)
			if err != nil {
				return nil, err
			}
			if hasDifference {
				diff.commentsToAdd = append(diff.commentsToAdd, commentDiff)
			}
		} else {
			diff.toCreate = append(diff.toCreate, expectedMirror)
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

func getViewerLogin(ctx context.Context, client *graphql.Client) (string, error) {
	vars := map[string]interface{}{}
	var data struct {
		Viewer struct {
			Login string `json:"login"`
		} `json:"viewer"`
	}
	err := client.Query(ctx, getViewerLoginQuery, vars, &data)
	if err != nil {
		return "", err
	}
	return data.Viewer.Login, nil
}

func (d issueDiffer) diffIssueComments(botLogin string, actualIssue, expectedIssue issueNode) (*comments.CommentDiff, bool, error) {
	expectedComments := expectedIssue.Comments.Nodes
	actualComments := actualIssue.CommentNodesByAuthor(botLogin)
	if len(expectedComments) == len(actualComments) {
		return nil, false, nil
	}
	diff := &comments.CommentDiff{
		Client:        d.toClient,
		Stdout:        d.stdout,
		MirrorIssueID: actualIssue.ID,
	}
	for i := len(actualComments); i < len(expectedComments); i++ {
		mirrorComment, err := expectedComments[i].ToMirrorComment()
		if err != nil {
			return nil, false, err
		}
		diff.ToCreate = append(diff.ToCreate, mirrorComment)
	}
	return diff, true, nil
}
