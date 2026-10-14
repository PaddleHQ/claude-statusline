@cache_hit_rate
Feature: Cache hit rate
  The cache segment shows the share of prompt tokens read from the cache, in red below cache.warn.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: A hit rate below cache.warn is shown in red
    Given the session:
      """json
      {
        "context_window": {
          "current_usage": {
            "input_tokens": 50,
            "cache_creation_input_tokens": 50,
            "cache_read_input_tokens": 4
          }
        }
      }
      """
    And the config:
      """
      segments = cache
      """
    When the status line renders
    Then the output is:
      """
      ⧉ 3%
      rrrr
      """
