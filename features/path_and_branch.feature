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
