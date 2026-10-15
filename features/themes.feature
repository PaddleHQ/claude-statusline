@themes
Feature: Themes
  Theme files are loaded by name, absolute path, relative path or ~ path, and an unusable theme
  falls back to the built-in one without printing an error.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: A theme path starting with ~ is resolved against the home directory
    Given the session:
      """json
      {
        "rate_limits": {
          "five_hour": {
            "used_percentage": 60,
            "resets_at": 1760009000
          }
        }
      }
      """
    And the config:
      """
      theme = ~/mine/ocean.yaml
      segments = rate5h
      """
    And the file "mine/ocean.yaml":
      """
      colors:
        text: ""
        muted: bright-black
        good: cyan
        warn: ""
        bad: ""
        marker: ""
        path: ""
      bar:
        style: solid
        full: "#"
        empty: "."
        marker: "|"
        partials: ["1", "2", "3", "4", "5", "6", "7"]
      """
    When the status line renders
    Then the output is:
      """
      5h: ##########|#........ 60%
          cccccccccc ckkkkkkkk
      """

  Scenario: An unknown theme name falls back to the built-in theme
    Given the session:
      """json
      {
        "rate_limits": {
          "five_hour": {
            "used_percentage": 60,
            "resets_at": 1760009000
          }
        }
      }
      """
    And the config:
      """
      theme = sunset
      segments = rate5h
      """
    When the status line renders
    Then the output is:
      """
      5h: ██████████┃█░░░░░░░░ 60%
          ggggggggggBgdddddddd
      """
