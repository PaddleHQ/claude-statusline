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
      ▕ctx║║║━━━╸────── 37%
      12223331456777777 111
      
      1 = #4aa8ff
      2 = #d8dee9 bg:#4b5160
      3 = #4aa8ff bg:#4b5160
      4 = #66b5ff
      5 = #85c4ff
      6 = #a4d3ff
      7 = #3a3a3a
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
      ▕5h║║║━━━━━━━━━─┃───────── 45% · ▕7d║║║━━━━━━━━──┃───────── 40%
      1223331111114567B777777777 111 d 1223331111145677B777777777 111
      
      1 = #4aa8ff
      2 = #d8dee9 bg:#4b5160
      3 = #4aa8ff bg:#4b5160
      4 = #66b5ff
      5 = #85c4ff
      6 = #a4d3ff
      7 = #3a3a3a
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
      ▕5h║║║━━━━━━━━━━┃━━━━───── 75%
      1223331111111111B145677777 111
      
      1 = #b26bff
      2 = #d8dee9 bg:#4b5160
      3 = #b26bff bg:#4b5160
      4 = #be82ff
      5 = #cb9bff
      6 = #d8b5ff
      7 = #3a3a3a
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
      ▕5h║║║━━━━━━━━━━┃━━━━━━━╸─ 92%
      1223331111111111B111114567 111
      
      1 = #ff3b30
      2 = #d8dee9 bg:#4b5160
      3 = #ff3b30 bg:#4b5160
      4 = #ff5a51
      5 = #ff7b74
      6 = #ff9d97
      7 = #3a3a3a
      """
