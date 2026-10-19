@gradient_theme
Feature: Gradient theme
  The gradient theme draws bars as blocks with a colour ramp and a slash for the pace marker.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: The gradient theme draws the context bar with gradient blocks and a coloured ramp
    Given the session:
      """json
      {
        "context_window": {
          "used_percentage": 62
        }
      }
      """
    And the config:
      """
      theme: gradient
      segments: [ctx]
      """
    When the status line renders
    Then the output is:
      """
      ctx: ▰▰▰▰▰▰▱▱▱▱ 62%
           1234567777 yyy

      1 = #94e2ad
      2 = #aae2ad
      3 = #c0e2ad
      4 = #d7e2ae
      5 = #ede2ae
      6 = #f9d9af
      7 = #4a4a4a
      """

  Scenario: The gradient theme's wide rate-limit bars for both the five hour and seven day windows
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
      theme: gradient
      segments: [rate5h, rate7d]
      """
    When the status line renders
    Then the output is:
      """
      5h: ▰▰▰▰▰▰▰▰▰▱/▱▱▱▱▱▱▱▱▱ 45% · 7d: ▰▰▰▰▰▰▰▰▱▱/▱▱▱▱▱▱▱▱▱ 40%
          123456789aBaaaaaaaaa ggg d     12345678aaBaaaaaaaaa ggg

      1 = #94e2ad
      2 = #9ee2ad
      3 = #a9e2ad
      4 = #b3e2ad
      5 = #bee2ad
      6 = #c9e2ae
      7 = #d3e2ae
      8 = #dee2ae
      9 = #e9e2ae
      a = #4a4a4a
      """
