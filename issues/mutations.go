package issues

const createIssueMutation = `
mutation($repository_id: ID!, $title: String!, $body: String!) {
	createIssue(input: {
		repositoryId: $repository_id,
		title: $title,
		body: $body,
	}) {
		issue {
			id
			number
		}
	}
}
`
