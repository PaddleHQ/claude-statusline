@rate_bars
Feature: Rate-limit bars
  The five hour and seven day usage bars, their colour against the pace marker, and their width
  at different terminal widths.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario Outline: Usage of <used>% with half the window elapsed is drawn in <result>
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"rate_limits":{"five_hour":{"used_percentage":<used>,"resets_at":1760009000}},"session_id":"abc123"}
      """
    When the status line renders
    Then the output is:
      """
      <output>
      """

    Examples:
      | used | result                                     | output                                                                                      |
      | 60   | green                                      | Opus 4.7 \e[2m·\e[0m 5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[32m█\e[0m\e[2m░░░░░░░░\e[0m 60% |
      | 75   | yellow                                     | Opus 4.7 \e[2m·\e[0m 5h: \e[33m██████████\e[0m\e[1m┃\e[0m\e[33m████\e[0m\e[2m░░░░░\e[0m 75% |
      | 92   | red                                        | Opus 4.7 \e[2m·\e[0m 5h: \e[31m██████████\e[0m\e[1m┃\e[0m\e[31m███████▍\e[0m\e[2m░\e[0m 92% |
      | 5    | red, with the pace marker well ahead of it | Opus 4.7 \e[2m·\e[0m 5h: \e[31m█\e[0m\e[2m░░░░░░░░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 5%   |

  Scenario: Usage of 96% fills the bar and carries on past the pace marker
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"rate_limits":{"five_hour":{"used_percentage":96,"resets_at":1760001800}},"session_id":"abc123"}
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 \e[2m·\e[0m 5h: \e[31m██████████████████\e[0m\e[1m┃\e[0m\e[31m▎\e[0m 96%
      """

  Scenario Outline: At <columns> columns the usage bars use their <width> width
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}
      """
    When the status line renders at <columns> columns
    Then the output is:
      """
      <output>
      """

    Examples:
      | columns | width  | output                                                                                                                                                        |
      | 100     | narrow | Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%         |
      | 140     | medium | Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m██████▎\e[0m\e[1m┃\e[0m\e[2m░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m█████▋\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░\e[0m 40% |

  Scenario: A 10 column terminal wraps the segments onto rows and uses the narrow bar width from rate5h.width_narrow
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}
      """
    And the config:
      """
      rate5h.width_narrow = 3
      """
    When the status line renders at 10 columns
    Then the output is:
      """
      Opus 4.7 [high]
      5h: \e[32m█▊\e[0m\e[1m┃\e[0m\e[2m░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%
      """

  Scenario: A rate-limit window whose reset time has passed is hidden
    Given the session:
      """json
      {"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1759999900},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}
      """
    When the status line renders
    Then the output is:
      """
      Opus 4.7 [high] \e[2m·\e[0m 7d: \e[32m████████\e[0m\e[2m░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 40%
      """
