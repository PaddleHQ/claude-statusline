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

  Scenario: The gradient theme at 95% context use
    Given the session:
      """json
      {"context_window":{"used_percentage":95}}
      """
    And the config:
      """
      theme = gradient
      segments = ctx
      """
    When the status line renders
    Then the output is:
      """
      ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;237;226;174m▰\e[0m\e[38;2;249;217;175m▰\e[0m\e[38;2;248;198;173m▰\e[0m\e[38;2;246;178;172m▰\e[0m\e[38;2;245;159;170m▰\e[0m\e[38;2;243;139;168m▰\e[0m \e[31m95%\e[0m
      """

  Scenario: The gradient theme at 0% context use draws an entirely empty bar
    Given the session:
      """json
      {"context_window":{"used_percentage":0}}
      """
    And the config:
      """
      theme = gradient
      segments = ctx
      """
    When the status line renders
    Then the output is:
      """
      ctx: \e[38;2;74;74;74m▱▱▱▱▱▱▱▱▱▱\e[0m \e[32m0%\e[0m
      """

  Scenario: The gradient theme's context bar at 100 columns
    Given the session:
      """json
      {"context_window":{"used_percentage":71}}
      """
    And the config:
      """
      theme = gradient
      segments = ctx
      """
    When the status line renders at 100 columns
    Then the output is:
      """
      ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;188;226;173m▰\e[0m\e[38;2;228;226;174m▰\e[0m\e[38;2;248;209;174m▰\e[0m\e[38;2;74;74;74m▱▱\e[0m \e[33m71%\e[0m
      """

  Scenario: The gradient theme's rate-limit bars at 140 columns, with a pace marker slash
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}
      """
    And the config:
      """
      theme = gradient
      segments = rate5h rate7d
      """
    When the status line renders at 140 columns
    Then the output is:
      """
      5h: \e[38;2;148;226;173m▰\e[0m\e[38;2;163;226;173m▰\e[0m\e[38;2;179;226;173m▰\e[0m\e[38;2;194;226;173m▰\e[0m\e[38;2;210;226;174m▰\e[0m\e[38;2;225;226;174m▰\e[0m\e[38;2;74;74;74m▱\e[0m\e[1m/\e[0m\e[38;2;74;74;74m▱▱▱▱▱▱\e[0m \e[32m45%\e[0m \e[2m·\e[0m 7d: \e[38;2;148;226;173m▰\e[0m\e[38;2;163;226;173m▰\e[0m\e[38;2;179;226;173m▰\e[0m\e[38;2;194;226;173m▰\e[0m\e[38;2;210;226;174m▰\e[0m\e[38;2;225;226;174m▰\e[0m\e[38;2;74;74;74m▱\e[0m\e[1m/\e[0m\e[38;2;74;74;74m▱▱▱▱▱▱\e[0m \e[32m40%\e[0m
      """

  Scenario: The gradient theme's rate-limit bars at 100 columns
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}
      """
    And the config:
      """
      theme = gradient
      segments = rate5h rate7d
      """
    When the status line renders at 100 columns
    Then the output is:
      """
      5h: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;237;226;174m▰\e[0m\e[1m/\e[0m\e[38;2;74;74;74m▱▱▱▱\e[0m \e[32m45%\e[0m \e[2m·\e[0m 7d: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;74;74;74m▱\e[0m\e[1m/\e[0m\e[38;2;74;74;74m▱▱▱▱\e[0m \e[32m40%\e[0m
      """

  Scenario: The gradient theme with NO_COLOR keeps the block characters but drops every colour
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":75,"resets_at":1760009000}},"context_window":{"used_percentage":62}}
      """
    And the config:
      """
      theme = gradient
      segments = ctx rate5h
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the output is:
      """
      ctx: ▰▰▰▰▰▰▱▱▱▱ 62% · 5h: ▰▰▰▰▰▰▰▰▰▰/▰▰▰▰▱▱▱▱▱ 75%
      """

  Scenario: The gradient theme at 92% usage with half the window elapsed
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":92,"resets_at":1760009000}},"session_id":"abc123"}
      """
    And the config:
      """
      theme = gradient
      segments = rate5h
      """
    When the status line renders
    Then the output is:
      """
      5h: \e[38;2;148;226;173m▰\e[0m\e[38;2;158;226;173m▰\e[0m\e[38;2;169;226;173m▰\e[0m\e[38;2;179;226;173m▰\e[0m\e[38;2;190;226;173m▰\e[0m\e[38;2;201;226;174m▰\e[0m\e[38;2;211;226;174m▰\e[0m\e[38;2;222;226;174m▰\e[0m\e[38;2;233;226;174m▰\e[0m\e[38;2;243;226;174m▰\e[0m\e[1m/\e[0m\e[38;2;249;213;174m▰\e[0m\e[38;2;248;204;174m▰\e[0m\e[38;2;247;194;173m▰\e[0m\e[38;2;247;185;172m▰\e[0m\e[38;2;246;176;171m▰\e[0m\e[38;2;245;167;171m▰\e[0m\e[38;2;245;158;170m▰\e[0m\e[38;2;74;74;74m▱▱\e[0m \e[31m92%\e[0m
      """

  Scenario: The gradient theme at 60% usage with half the window elapsed
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}
      """
    And the config:
      """
      theme = gradient
      segments = rate5h
      """
    When the status line renders
    Then the output is:
      """
      5h: \e[38;2;148;226;173m▰\e[0m\e[38;2;158;226;173m▰\e[0m\e[38;2;169;226;173m▰\e[0m\e[38;2;179;226;173m▰\e[0m\e[38;2;190;226;173m▰\e[0m\e[38;2;201;226;174m▰\e[0m\e[38;2;211;226;174m▰\e[0m\e[38;2;222;226;174m▰\e[0m\e[38;2;233;226;174m▰\e[0m\e[38;2;243;226;174m▰\e[0m\e[1m/\e[0m\e[38;2;249;213;174m▰\e[0m\e[38;2;74;74;74m▱▱▱▱▱▱▱▱\e[0m \e[32m60%\e[0m
      """

  Scenario: The gradient theme at 75% usage with half the window elapsed
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":75,"resets_at":1760009000}},"session_id":"abc123"}
      """
    And the config:
      """
      theme = gradient
      segments = rate5h
      """
    When the status line renders
    Then the output is:
      """
      5h: \e[38;2;148;226;173m▰\e[0m\e[38;2;158;226;173m▰\e[0m\e[38;2;169;226;173m▰\e[0m\e[38;2;179;226;173m▰\e[0m\e[38;2;190;226;173m▰\e[0m\e[38;2;201;226;174m▰\e[0m\e[38;2;211;226;174m▰\e[0m\e[38;2;222;226;174m▰\e[0m\e[38;2;233;226;174m▰\e[0m\e[38;2;243;226;174m▰\e[0m\e[1m/\e[0m\e[38;2;249;213;174m▰\e[0m\e[38;2;248;204;174m▰\e[0m\e[38;2;247;194;173m▰\e[0m\e[38;2;247;185;172m▰\e[0m\e[38;2;74;74;74m▱▱▱▱▱\e[0m \e[33m75%\e[0m
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
