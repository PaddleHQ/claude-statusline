@path_and_branch
Feature: Path and git branch
  The path segment abbreviates the home directory, can strip prefixes and shows the git branch,
  found by reading .git files without running git.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: The branch is found by walking up from a nested working directory to the repository's .git
    Given the session:
      """json
      {
        "cwd": "@TMP@/repo/pkg/sub"
      }
      """
    And the config:
      """
      segments: [path]
      """
    And the environment variable "NO_COLOR" is "1"
    And the file "repo/.git/HEAD":
      """
      ref: refs/heads/main
      """
    And the directory "repo/pkg/sub"
    When the status line renders
    Then the text is:
      """
      ~/repo/pkg/sub (main)
      """

  Scenario: A worktree under a configured directory shows the project name with the worktree emoji and colour
    Given the session:
      """json
      {
        "cwd": "@TMP@/repo/.worktrees/statusline_cost"
      }
      """
    And the config:
      """
      segments: [path]
      path:
        worktrees: [.worktrees]
        emoji: true
      """
    And the file "repo/.git/HEAD":
      """
      ref: refs/heads/main
      """
    And the file "repo/.git/worktrees/statusline_cost/HEAD":
      """
      ref: refs/heads/statusline_cost
      """
    And the file "repo/.git/worktrees/statusline_cost/commondir":
      """
      ../..
      """
    And the file "repo/.worktrees/statusline_cost/.git":
      """
      gitdir: @TMP@/repo/.git/worktrees/statusline_cost
      """
    When the status line renders
    Then the output is:
      """
      repo (🌳 statusline_cost)
      1111113_33333333333333331

      1 = bold 256:33
      3 = bold 256:208
      """
