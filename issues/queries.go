package issues

const getIssueQuery = `
query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
		id
    issues(first: 100) {
      nodes {
				number
        title
        body
				createdAt
				url
        author {
          login
          url
					avatarUrl
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
