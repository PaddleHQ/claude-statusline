@config_error
Feature: A config that cannot be used
  When the config file cannot be used, the status line still shows the defaults and puts one row above it
  that names the file, the line the parser reached and why, so the setting that is ignored is not a mystery.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: A config that is not valid YAML shows one error row above the default line
    Given the session:
      """json
      {
        "model": {
          "display_name": "Opus 4.7"
        }
      }
      """
    And the config:
      """
      hooks:
        ttl: 60
       timeout: 2
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the text is:
      """
      ~/claude-statusline/config.yaml:2 did not find expected key · run /statusline:config-check
      Opus 4.7
      """

  Scenario: The error row is drawn in red
    Given the session:
      """json
      {
        "model": {
          "display_name": "Opus 4.7"
        }
      }
      """
    And the config:
      """
      hooks:
        ttl: 60
       timeout: 2
      """
    When the status line renders
    Then the output is:
      """
      ~/claude-statusline/config.yaml:2 did not find expected key · run /statusline:config-check
      rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr
      Opus 4.7

      """
