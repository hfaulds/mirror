package comments

const addCommentMutation = `
mutation($subject_id: ID!, $body: String!) {
	addComment(input: { subjectId: $subject_id, body: $body}) {
		commentEdge {
			node {
				id
				body
			}
		}
	}
}
`
