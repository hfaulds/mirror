package issues

const getIssueQuery = `
query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    id
    issues(first: 100) {
      nodes {
        id
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
            createdAt
            url
            author {
              avatarUrl
              login
              url
            }
          }
        }
      }
    }
  }
}
`

const getViewerLoginQuery = `
query {
 viewer() {
  login
 }
}
`
