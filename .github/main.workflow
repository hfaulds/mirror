workflow "Sync Issues" {
  resolves = ["Sync Issues Action"]
  on = "issues"
}

workflow "Sync Issue Comments" {
  on = "issue_comment"
  resolves = [ "Sync Issues Action" ]
}

action "Sync Issues Action" {
  uses = "./"
  args = "mirror issues --from hfaulds/mirror --to hfaulds/mirror-mirror"
  secrets = [
    "TO_TOKEN",
    "GITHUB_TOKEN",
  ]
}
