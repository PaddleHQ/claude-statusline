@path_and_branch
Feature: Path and git branch
  The path segment abbreviates the home directory, can strip prefixes and shows the git branch,
  found by reading .git files without running git.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: A path under the home directory starts with ~
    Given the session:
      """json
      {"cwd":"@TMP@/work/app"}
      """
    And the config:
      """
      segments = path
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the output is:
      """
      ~/work/app
      """

  Scenario: The home directory itself is shown as ~
    Given the session:
      """json
      {"cwd":"@TMP@"}
      """
    And the config:
      """
      segments = path
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the output is:
      """
      ~
      """

  Scenario: A directory that merely starts with the home directory's name is not abbreviated to ~
    Given the session:
      """json
      {"cwd":"@TMP@-other/app"}
      """
    And the config:
      """
      segments = path
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the output is:
      """
      @TMP@-other/app
      """

  Scenario: A path outside the home directory is shown in full
    Given the session:
      """json
      {"cwd":"/srv/outside/app"}
      """
    And the config:
      """
      segments = path
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the output is:
      """
      /srv/outside/app
      """

  Scenario: The workspace directory takes precedence over the working directory
    Given the session:
      """json
      {"cwd":"@TMP@/old","workspace":{"current_dir":"@TMP@/new"}}
      """
    And the config:
      """
      segments = path
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the output is:
      """
      ~/new
      """

  Scenario: The path segment is drawn in bold blue
    Given the session:
      """json
      {"cwd":"@TMP@/work/app"}
      """
    And the config:
      """
      segments = path
      """
    When the status line renders
    Then the output is:
      """
      \e[1m\e[38;5;33m~/work/app\e[0m
      """

  Scenario Outline: path.strip <rule>
    Given the session:
      """json
      {"cwd":"<cwd>"}
      """
    And the config:
      """
      segments = path
      path.strip = <strip>
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the output is:
      """
      <output>
      """

    Examples:
      | cwd                 | strip                   | rule                                                                 | output         |
      | /srv/work/app       | /srv/work               | removes an absolute path prefix                                      | app            |
      | @TMP@/code/team/app | ~/code ~/code/team /srv | removes the longest of several matching prefixes                     | app            |
      | @TMP@/code/team/app | ~/projects ~/code       | removes the first listed prefix that matches                         | team/app       |
      | @TMP@/codebase/app  | ~/code                  | does not remove a prefix that ends part-way through a directory name | ~/codebase/app |

  Scenario: A detached HEAD shows the first seven characters of the commit hash in place of a branch name
    Given the session:
      """json
      {"cwd":"@TMP@/repo"}
      """
    And the config:
      """
      segments = path
      """
    And the environment variable "NO_COLOR" is "1"
    And the file "repo/.git/HEAD":
      """
      0123456789abcdef0123456789abcdef01234567
      """
    When the status line renders
    Then the output is:
      """
      ~/repo (0123456)
      """

  Scenario: path.branch = false hides the branch even inside a git repository
    Given the session:
      """json
      {"cwd":"@TMP@/repo"}
      """
    And the config:
      """
      segments = path
      path.branch = false
      """
    And the environment variable "NO_COLOR" is "1"
    And the file "repo/.git/HEAD":
      """
      ref: refs/heads/main
      """
    When the status line renders
    Then the output is:
      """
      ~/repo
      """

  Scenario: The branch is found by walking up from a nested working directory to the repository's .git
    Given the session:
      """json
      {"cwd":"@TMP@/repo/pkg/sub"}
      """
    And the config:
      """
      segments = path
      """
    And the environment variable "NO_COLOR" is "1"
    And the file "repo/.git/HEAD":
      """
      ref: refs/heads/main
      """
    And the directory "repo/pkg/sub"
    When the status line renders
    Then the output is:
      """
      ~/repo/pkg/sub (main)
      """

  Scenario: A .git directory with no readable HEAD file shows the path without a branch
    Given the session:
      """json
      {"cwd":"@TMP@/repo"}
      """
    And the config:
      """
      segments = path
      """
    And the environment variable "NO_COLOR" is "1"
    And the file "repo/.git/config":
      """
      x
      """
    When the status line renders
    Then the output is:
      """
      ~/repo
      """

  Scenario: A linked worktree, where .git is a file pointing at the real git directory, still shows its branch
    Given the session:
      """json
      {"cwd":"@TMP@/wt"}
      """
    And the config:
      """
      segments = path
      """
    And the environment variable "NO_COLOR" is "1"
    And the file "wt/.git":
      """
      gitdir: ../main/.git/worktrees/wt
      """
    And the file "main/.git/worktrees/wt/HEAD":
      """
      ref: refs/heads/topic
      """
    When the status line renders
    Then the output is:
      """
      ~/wt (topic)
      """
