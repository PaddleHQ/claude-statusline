@saber_theme
Feature: Saber theme
  The saber theme draws bars as a lightsaber blade coloured by pace that brightens towards the tip.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: The saber theme draws the context bar as a blade with a half-lit tip
    Given the session:
      """json
      {
        "context_window": {
          "used_percentage": 37
        }
      }
      """
    And the config:
      """
      theme: saber
      segments: [ctx]
      """
    When the status line renders
    Then the output is:
      """
      ctx: ━━━╸────── 37%
           1234555555 111

      1 = #4aa8ff
      2 = #66b5ff
      3 = #85c4ff
      4 = #a4d3ff
      5 = #3a3a3a
      """

  Scenario: The saber theme's rate-limit bars take their blade colour from the pace
    Given the session:
      """json
      {
        "rate_limits": {
          "five_hour": {
            "used_percentage": 45.4,
            "resets_at": 1760009000
          },
          "seven_day": {
            "used_percentage": 40,
            "resets_at": 1760302400
          }
        }
      }
      """
    And the config:
      """
      theme: saber
      segments: [rate5h, rate7d]
      """
    When the status line renders
    Then the output is:
      """
      5h: ━━━━━━━━━─┃───────── 45% · 7d: ━━━━━━━━──┃───────── 40%
          1111112345B555555555 111 d     1111123455B555555555 111

      1 = #4aa8ff
      2 = #66b5ff
      3 = #85c4ff
      4 = #a4d3ff
      5 = #3a3a3a
      """

  Scenario: The saber theme's blade turns purple when usage runs ahead of the pace (75%)
    Given the session:
      """json
      {
        "rate_limits": {
          "five_hour": {
            "used_percentage": 75,
            "resets_at": 1760009000
          }
        }
      }
      """
    And the config:
      """
      theme: saber
      segments: [rate5h]
      """
    When the status line renders
    Then the output is:
      """
      5h: ━━━━━━━━━━┃━━━━───── 75%
          1111111111B123455555 111

      1 = #b26bff
      2 = #be82ff
      3 = #cb9bff
      4 = #d8b5ff
      5 = #3a3a3a
      """

  Scenario: The saber theme's blade turns red when usage runs ahead of the pace (92%)
    Given the session:
      """json
      {
        "rate_limits": {
          "five_hour": {
            "used_percentage": 92,
            "resets_at": 1760009000
          }
        }
      }
      """
    And the config:
      """
      theme: saber
      segments: [rate5h]
      """
    When the status line renders
    Then the output is:
      """
      5h: ━━━━━━━━━━┃━━━━━━━╸─ 92%
          1111111111B111112345 111

      1 = #ff3b30
      2 = #ff5a51
      3 = #ff7b74
      4 = #ff9d97
      5 = #3a3a3a
      """
