@themes
Feature: Themes
  Theme files are loaded by name, absolute path, relative path or ~ path, and an unusable theme
  falls back to the built-in one without printing an error.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: A theme file with an unknown colour name is rejected and the built-in theme is used
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}
      """
    And the config:
      """
      theme = ~/bad.yaml
      segments = rate5h
      """
    And the file "bad.yaml":
      """
      colors:
        text: ""
        muted: dim
        good: chartreuse
        warn: yellow
        bad: red
        marker: bold
        path: bold 33
      bar:
        style: gradient
        full: "▰"
        partials: ["▏", "▎", "▍", "▌", "▋", "▊", "▉"]
        empty: "▱"
        marker: "/"
        ramp: ["#94e2ad", "#f9e2af", "#f38ba8"]
      """
    When the status line renders
    Then the output is:
      """
      5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[32m█\e[0m\e[2m░░░░░░░░\e[0m 60%
      """

  Scenario: A theme file whose gradient ramp is invalid is rejected and the built-in theme is used
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}
      """
    And the config:
      """
      theme = ~/bad.yaml
      segments = rate5h
      """
    And the file "bad.yaml":
      """
      colors:
        text: ""
        muted: dim
        good: green
        warn: yellow
        bad: red
        marker: bold
        path: bold 33
      bar:
        style: gradient
        full: "▰"
        partials: ["▏", "▎", "▍", "▌", "▋", "▊", "▉"]
        empty: "▱"
        marker: "/"
        ramp: ["#fff", "#000000", "#000000"]
      """
    When the status line renders
    Then the output is:
      """
      5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[32m█\e[0m\e[2m░░░░░░░░\e[0m 60%
      """

  Scenario: A theme file that is not valid YAML is rejected and the built-in theme is used
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}
      """
    And the config:
      """
      theme = ~/bad.yaml
      segments = rate5h
      """
    And the file "bad.yaml":
      """
      colors: [unclosed
      """
    When the status line renders
    Then the output is:
      """
      5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[32m█\e[0m\e[2m░░░░░░░░\e[0m 60%
      """

  Scenario: theme = default is accepted and draws the built-in look
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}
      """
    And the config:
      """
      theme = default
      segments = rate5h
      """
    When the status line renders
    Then the output is:
      """
      5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[32m█\e[0m\e[2m░░░░░░░░\e[0m 60%
      """

  Scenario: A theme file named by an absolute path is loaded
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}
      """
    And the config:
      """
      theme = @TMP@/claude-statusline/ocean.yaml
      segments = rate5h
      """
    And the file "claude-statusline/ocean.yaml":
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
      5h: \e[36m##########\e[0m|\e[36m#\e[0m\e[90m........\e[0m 60%
      """

  Scenario: A relative theme path is resolved against the config directory
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}
      """
    And the config:
      """
      theme = themes/ocean.yaml
      segments = rate5h
      """
    And the file "claude-statusline/themes/ocean.yaml":
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
      5h: \e[36m##########\e[0m|\e[36m#\e[0m\e[90m........\e[0m 60%
      """

  Scenario: A theme path starting with ~ is resolved against the home directory
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}
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
      5h: \e[36m##########\e[0m|\e[36m#\e[0m\e[90m........\e[0m 60%
      """

  Scenario: A theme file that does not exist falls back to the built-in theme
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}
      """
    And the config:
      """
      theme = ~/nope.yaml
      segments = rate5h
      """
    When the status line renders
    Then the output is:
      """
      5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[32m█\e[0m\e[2m░░░░░░░░\e[0m 60%
      """

  Scenario: A theme file with an unknown key is rejected and the built-in theme is used
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}
      """
    And the config:
      """
      theme = ~/bad.yaml
      segments = rate5h
      """
    And the file "bad.yaml":
      """
      colours:
        text: ""
        muted: dim
        good: green
        warn: yellow
        bad: red
        marker: bold
        path: bold 33
      bar:
        style: gradient
        full: "▰"
        partials: ["▏", "▎", "▍", "▌", "▋", "▊", "▉"]
        empty: "▱"
        marker: "/"
        ramp: ["#94e2ad", "#f9e2af", "#f38ba8"]
      """
    When the status line renders
    Then the output is:
      """
      5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[32m█\e[0m\e[2m░░░░░░░░\e[0m 60%
      """

  Scenario: An unknown theme name falls back to the built-in theme
    Given the session:
      """json
      {"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}
      """
    And the config:
      """
      theme = sunset
      segments = rate5h
      """
    When the status line renders
    Then the output is:
      """
      5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[32m█\e[0m\e[2m░░░░░░░░\e[0m 60%
      """
