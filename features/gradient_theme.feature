@gradient_theme
Feature: Gradient theme
  The gradient theme draws bars as blocks with a colour ramp and a slash for the pace marker.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: The gradient theme draws the context bar with gradient blocks and a coloured ramp
    Given the session:
      """json
      {"context_window":{"used_percentage":62}}
      """
    And the config:
      """
      theme = gradient
      segments = ctx
      """
    When the status line renders
    Then the output is:
      """
      ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;237;226;174m▰\e[0m\e[38;2;249;217;175m▰\e[0m\e[38;2;74;74;74m▱▱▱▱\e[0m \e[33m62%\e[0m
      """

  Scenario: The gradient theme's wide rate-limit bars for both the five hour and seven day windows
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":45.4,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}
      """
    And the config:
      """
      theme = gradient
      segments = rate5h rate7d
      """
    When the status line renders
    Then the output is:
      """
      5h: \e[38;2;148;226;173m▰\e[0m\e[38;2;158;226;173m▰\e[0m\e[38;2;169;226;173m▰\e[0m\e[38;2;179;226;173m▰\e[0m\e[38;2;190;226;173m▰\e[0m\e[38;2;201;226;174m▰\e[0m\e[38;2;211;226;174m▰\e[0m\e[38;2;222;226;174m▰\e[0m\e[38;2;233;226;174m▰\e[0m\e[38;2;74;74;74m▱\e[0m\e[1m/\e[0m\e[38;2;74;74;74m▱▱▱▱▱▱▱▱▱\e[0m \e[32m45%\e[0m \e[2m·\e[0m 7d: \e[38;2;148;226;173m▰\e[0m\e[38;2;158;226;173m▰\e[0m\e[38;2;169;226;173m▰\e[0m\e[38;2;179;226;173m▰\e[0m\e[38;2;190;226;173m▰\e[0m\e[38;2;201;226;174m▰\e[0m\e[38;2;211;226;174m▰\e[0m\e[38;2;222;226;174m▰\e[0m\e[38;2;74;74;74m▱▱\e[0m\e[1m/\e[0m\e[38;2;74;74;74m▱▱▱▱▱▱▱▱▱\e[0m \e[32m40%\e[0m
      """
