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