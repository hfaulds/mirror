workflow "Sync Issues" {
  on = "issue"
  resolves = [ "Sync Issues Action" ]
}

action "Sync Issues Action" {
  uses = "."
  args = "mirror issues --from hfaulds/mirror --to hfaulds/mirror-mirror"
  secrets = ["GITHUB_TO_TOKEN"]
}
