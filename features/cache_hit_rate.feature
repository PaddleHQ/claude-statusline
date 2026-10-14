@cache_hit_rate
Feature: Cache hit rate
  The cache segment shows the share of prompt tokens read from the cache, in red below cache.warn.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario Outline: A cache hit rate of <rate>% is shown <style>
    Given the session:
      """json
      {"context_window":{"current_usage":{"input_tokens":<input>,"cache_creation_input_tokens":<created>,"cache_read_input_tokens":<read>}}}
      """
    And the config:
      """
      segments = cache
      """
    When the status line renders
    Then the output is:
      """
      <output>
      """

    Examples:
      | input | created | read | rate | style          | output          |
      | 10    | 10      | 80   | 80   | without colour | ⧉ 80%           |
      | 95    | 0       | 5    | 5    | without colour | ⧉ 5%            |
      | 50    | 50      | 4    | 3    | in red         | \e[31m⧉ 3%\e[0m |

  Scenario: cache.warn = 90 turns an 80% hit rate red
    Given the session:
      """json
      {"context_window":{"current_usage":{"input_tokens":10,"cache_creation_input_tokens":10,"cache_read_input_tokens":80}}}
      """
    And the config:
      """
      segments = cache
      cache.warn = 90
      """
    When the status line renders
    Then the output is:
      """
      \e[31m⧉ 80%\e[0m
      """

  Scenario: The cache segment is hidden when the token counts add up to zero, so there is no rate to compute
    Given the session:
      """json
      {"context_window":{"current_usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
      """
    And the config:
      """
      segments = cache
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders
    Then the output is:
      """
      """
