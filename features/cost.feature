@cost
Feature: Session cost
  The cost segment shows what the session has cost so far in US dollars, amber from cost.warn and red from cost.bad.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: A cost between cost.warn and cost.bad is shown in amber after the cache rate
    Given the session:
      """json
      {
        "context_window": {
          "current_usage": {
            "input_tokens": 10,
            "cache_creation_input_tokens": 10,
            "cache_read_input_tokens": 80
          }
        },
        "cost": {
          "total_cost_usd": 7.456
        }
      }
      """
    And the config:
      """
      segments: [cache, cost]
      segments_excluded: []
      """
    When the status line renders
    Then the output is:
      """
      ⧉ 80% · $7.46
            d yyyyy
      """
