@capsule_theme
Feature: Capsule theme
  The capsule theme draws rounded Nerd Font bars coloured by pace when icons are nerd, and plain capsule glyphs otherwise.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: The capsule theme with Nerd icons colours the bar by pace
    Given the session:
      """json
      {
        "rate_limits": {
          "five_hour": {
            "used_percentage": 45.4,
            "resets_at": 1760009000
          }
        }
      }
      """
    And the config:
      """
      theme: capsule
      icons: nerd
      segments: [rate5h]
      """
    When the status line renders
    Then the output is:
      """
       5h  45%
           ggggggggg1B111111111 ggg

      1 = #5a5a5a
      """

  Scenario: The capsule theme without Nerd icons draws plain capsule glyphs in whole cells
    Given the session:
      """json
      {
        "rate_limits": {
          "five_hour": {
            "used_percentage": 45.4,
            "resets_at": 1760009000
          }
        }
      }
      """
    And the config:
      """
      theme: capsule
      icons: none
      segments: [rate5h]
      """
    When the status line renders
    Then the output is:
      """
      5h: ▮▮▮▮▮▮▮▮▮▯┃▯▯▯▯▯▯▯▯▯ 45%
          gggggggggdBddddddddd
      """

  Scenario: The capsule theme without colour
    Given the session:
      """json
      {
        "rate_limits": {
          "five_hour": {
            "used_percentage": 45.4,
            "resets_at": 1760009000
          }
        }
      }
      """
    And the config:
      """
      theme: capsule
      icons: nerd
      segments: [rate5h]
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the text is:
      """
       5h  45%
      """
