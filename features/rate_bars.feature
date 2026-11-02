@rate_bars
Feature: Rate-limit bars
  The five hour and seven day usage bars: the colour follows the pace marker.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: Usage of 75% with half the window elapsed is drawn in yellow
    Given the session:
      """json
      {
        "model": {
          "display_name": "Opus 4.7"
        },
        "rate_limits": {
          "five_hour": {
            "used_percentage": 75,
            "resets_at": 1760009000
          }
        }
      }
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 · 5h: ██████████┃████░░░░░ 75%
               d     yyyyyyyyyyByyyyddddd
      """

  Scenario: The rate-bars command prints both windows as JSON for other tools to draw
    Given the rate-limit cache:
      """
      {"five_hour":{"used":75,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """
    When the rate bars render for 4 to 5 cells
    Then the raw output is:
      """
      {"rate5h":{"ok":true,"percent":"75%","bars":{"4":"\u001b[33m██\u001b[0m\u001b[1m┃\u001b[0m\u001b[2m░\u001b[0m","5":"\u001b[33m██\u001b[0m\u001b[1m┃\u001b[0m\u001b[33m▊\u001b[0m\u001b[2m░\u001b[0m"}},"rate7d":{"ok":true,"percent":"40%","bars":{"4":"\u001b[32m█▋\u001b[0m\u001b[1m┃\u001b[0m\u001b[2m░\u001b[0m","5":"\u001b[32m██\u001b[0m\u001b[1m┃\u001b[0m\u001b[2m░░\u001b[0m"}}}
      """

  Scenario: With a hilt theme the rate-bars command also prints each window's label drawn as the hilt
    Given the config:
      """
      theme: saber
      """
    And the rate-limit cache:
      """
      {"five_hour":{"used":75,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}
      """
    When the rate bars render for 4 to 4 cells
    Then the raw output is:
      """
      {"rate5h":{"ok":true,"label":"\u001b[38;2;178;107;255m\u001b[48;2;75;81;96m▏\u001b[0m\u001b[48;2;75;81;96m\u001b[38;2;216;222;233m5h\u001b[0m\u001b[38;2;178;107;255m\u001b[48;2;75;81;96m║║║\u001b[0m","percent":"\u001b[38;2;178;107;255m75%\u001b[0m","bars":{"4":"\u001b[38;2;190;130;255m━\u001b[0m\u001b[38;2;203;155;255m━\u001b[0m\u001b[1m┃\u001b[0m\u001b[38;2;58;58;58m─\u001b[0m"}},"rate7d":{"ok":true,"label":"\u001b[38;2;74;168;255m\u001b[48;2;75;81;96m▏\u001b[0m\u001b[48;2;75;81;96m\u001b[38;2;216;222;233m7d\u001b[0m\u001b[38;2;74;168;255m\u001b[48;2;75;81;96m║║║\u001b[0m","percent":"\u001b[38;2;74;168;255m40%\u001b[0m","bars":{"4":"\u001b[38;2;119;189;255m━\u001b[0m\u001b[38;2;164;211;255m╸\u001b[0m\u001b[1m┃\u001b[0m\u001b[38;2;58;58;58m─\u001b[0m"}}}
      """
