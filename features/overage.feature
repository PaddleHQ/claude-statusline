@overage
Feature: Extra-usage spend
  The overage segment shows the extra-usage spend this calendar month. Logins that report a spend limit show it, in amber once anything is spent.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: A reported spend limit is shown as dollars used of the limit, in amber, after the cost
    Given the session:
      """json
      {
        "cost": {
          "total_cost_usd": 7.456
        },
        "rate_limits": {
          "spend_limit": {
            "used_percentage": 25,
            "resets_at": 1762000000,
            "used_usd": 12.5,
            "limit_usd": 50
          }
        }
      }
      """
    And the config:
      """
      segments: [cost, overage]
      segments_excluded: []
      """
    When the status line renders
    Then the output is:
      """
      $7.46 · » $12.50/$50
      yyyyy d yyyyyyyyyyyy
      """
