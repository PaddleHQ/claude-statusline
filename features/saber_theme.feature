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
      █▀ ctx ▕▕▕━━━╸────── 37%
      11222223334567888888 444
      
      1 = #4b5160
      2 = #d8dee9 bg:#4b5160
      3 = #9aa3b5 bg:#4b5160
      4 = #4aa8ff
      5 = #66b5ff
      6 = #85c4ff
      7 = #a4d3ff
      8 = #3a3a3a
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
      █▀ 5h ▕▕▕━━━━━━━━━─┃───────── 45% · █▀ 7d ▕▕▕━━━━━━━━──┃───────── 40%
      1122223334444445678B888888888 444 d 1122223334444456788B888888888 444
      
      1 = #4b5160
      2 = #d8dee9 bg:#4b5160
      3 = #9aa3b5 bg:#4b5160
      4 = #4aa8ff
      5 = #66b5ff
      6 = #85c4ff
      7 = #a4d3ff
      8 = #3a3a3a
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
      █▀ 5h ▕▕▕━━━━━━━━━━┃━━━━───── 75%
      1122223334444444444B456788888 444
      
      1 = #4b5160
      2 = #d8dee9 bg:#4b5160
      3 = #9aa3b5 bg:#4b5160
      4 = #b26bff
      5 = #be82ff
      6 = #cb9bff
      7 = #d8b5ff
      8 = #3a3a3a
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
      █▀ 5h ▕▕▕━━━━━━━━━━┃━━━━━━━╸─ 92%
      1122223334444444444B444445678 444
      
      1 = #4b5160
      2 = #d8dee9 bg:#4b5160
      3 = #9aa3b5 bg:#4b5160
      4 = #ff3b30
      5 = #ff5a51
      6 = #ff7b74
      7 = #ff9d97
      8 = #3a3a3a
      """
