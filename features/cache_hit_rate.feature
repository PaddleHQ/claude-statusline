@cache_hit_rate
Feature: Cache hit rate
  The cache segment shows the share of prompt tokens read from the cache, in red below cache.warn, or "cold" once the cache has expired.

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
      segments: [cache]
      """
    When the status line renders
    Then the output is:
      """
      ⧉ 3%
      rrrr
      """

  Scenario: A resumed session whose cache has expired reads cold, whatever the last hit rate was
    Given the session:
      """json
      {
        "transcript_path": "@TMP@/transcript.jsonl",
        "context_window": {
          "current_usage": {
            "input_tokens": 10,
            "cache_creation_input_tokens": 10,
            "cache_read_input_tokens": 80
          }
        }
      }
      """
    And the file "transcript.jsonl":
      """
      {"type":"assistant","timestamp":"2025-10-09T07:53:20Z","message":{"model":"claude-opus-4-7","usage":{"input_tokens":10,"cache_creation":{"ephemeral_1h_input_tokens":10,"ephemeral_5m_input_tokens":0}}}}
      """
    And the config:
      """
      segments: [cache]
      """
    When the status line renders
    Then the output is:
      """
      ⧉ cold
      rrrrrr
      """

  Scenario: A cache still within its hour shows the hit rate
    Given the session:
      """json
      {
        "transcript_path": "@TMP@/transcript.jsonl",
        "context_window": {
          "current_usage": {
            "input_tokens": 10,
            "cache_creation_input_tokens": 10,
            "cache_read_input_tokens": 80
          }
        }
      }
      """
    And the file "transcript.jsonl":
      """
      {"type":"assistant","timestamp":"2025-10-09T07:53:21Z","message":{"model":"claude-opus-4-7","usage":{"input_tokens":10,"cache_creation":{"ephemeral_1h_input_tokens":10,"ephemeral_5m_input_tokens":0}}}}
      """
    And the config:
      """
      segments: [cache]
      """
    When the status line renders
    Then the text is:
      """
      ⧉ 80%
      """
