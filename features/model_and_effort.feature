@model_and_effort
Feature: Model and effort
  The model segment shows the display name, followed by the effort level when there is one.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  Scenario: model.effort = false hides the effort level after the model name
    Given the session:
      """json
      {
        "model": {
          "display_name": "Opus 4.7"
        },
        "effort": {
          "level": "high"
        }
      }
      """
    And the config:
      """
      model:
        effort: false
      """
    When the status line renders
    Then the text is:
      """
      Opus 4.7
      """
