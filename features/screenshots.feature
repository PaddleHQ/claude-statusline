@screenshots
Feature: Documentation screenshots
  Each scenario saves what the status line prints as a screenshot, which go run ./tools/screenshots
  draws into docs/images. The images are checked against these scenarios by a test.

  Background:
    Given the environment variable "STATUSLINE_NOW" is "1760000000"
    And the environment variable "COLUMNS" is "200"

  @screenshot
  Scenario: The status line at 200 columns with every default segment, for the README
    Given the session:
      """json
      {
        "session_id": "demo",
        "model": {
          "id": "claude-opus-4-7",
          "display_name": "Opus 4.7"
        },
        "effort": {
          "level": "high"
        },
        "workspace": {
          "current_dir": "@TMP@/code/app"
        },
        "context_window": {
          "used_percentage": 42.4,
          "total_input_tokens": 1500,
          "total_output_tokens": 23400,
          "current_usage": {
            "input_tokens": 2000,
            "cache_creation_input_tokens": 1000,
            "cache_read_input_tokens": 17000
          }
        },
        "rate_limits": {
          "five_hour": {
            "used_percentage": 45.4,
            "resets_at": 1760009000
          },
          "seven_day": {
            "used_percentage": 62,
            "resets_at": 1760302400
          }
        },
        "transcript_path": "@TMP@/transcript.jsonl"
      }
      """
    And the file "code/app/.git/HEAD":
      """
      ref: refs/heads/feature/retry-backoff
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- review: Review a change for correctness, style and missing tests before it is merged\u000a- debug: Work out why a test fails or a service misbehaves, starting from the evidence\u000a- plan: Write an implementation plan for a multi-step change before touching code\u000a- release: Prepare release notes and check the version label on a pull request\u000a- docs: Write or restructure a repository's README and contributing guide\u000a"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      """
    When the status line renders at 200 columns
    Then the output is saved as the screenshot "hero"

  @screenshot
  Scenario: The status line in the gradient theme, for the themes section of the customising guide
    Given the session:
      """json
      {
        "session_id": "demo",
        "model": {
          "id": "claude-opus-4-7",
          "display_name": "Opus 4.7"
        },
        "effort": {
          "level": "high"
        },
        "workspace": {
          "current_dir": "@TMP@/code/app"
        },
        "context_window": {
          "used_percentage": 42.4,
          "total_input_tokens": 1500,
          "total_output_tokens": 23400,
          "current_usage": {
            "input_tokens": 2000,
            "cache_creation_input_tokens": 1000,
            "cache_read_input_tokens": 17000
          }
        },
        "rate_limits": {
          "five_hour": {
            "used_percentage": 45.4,
            "resets_at": 1760009000
          },
          "seven_day": {
            "used_percentage": 62,
            "resets_at": 1760302400
          }
        },
        "transcript_path": "@TMP@/transcript.jsonl"
      }
      """
    And the config:
      """
      theme: gradient
      segments: [path, model, ctx, tokens, cache, rate5h, rate7d]
      """
    And the file "code/app/.git/HEAD":
      """
      ref: refs/heads/feature/retry-backoff
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- review: Review a change for correctness, style and missing tests before it is merged\u000a- debug: Work out why a test fails or a service misbehaves, starting from the evidence\u000a- plan: Write an implementation plan for a multi-step change before touching code\u000a- release: Prepare release notes and check the version label on a pull request\u000a- docs: Write or restructure a repository's README and contributing guide\u000a"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      """
    When the status line renders at 200 columns
    Then the output is saved as the screenshot "gradient"

  @screenshot
  Scenario: The gradient theme's rate-limit bar with NO_COLOR, showing it still reads without colour
    Given the session:
      """json
      {
        "session_id": "demo",
        "rate_limits": {
          "five_hour": {
            "used_percentage": 75,
            "resets_at": 1760009000
          }
        }
      }
      """
    And the config:
      """
      theme: gradient
      segments: [rate5h]
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders at 200 columns
    Then the output is saved as the screenshot "gradient-no-color"

  @screenshot
  Scenario: The status line in the saber theme, for the themes section of the customising guide
    Given the session:
      """json
      {
        "session_id": "demo",
        "model": {
          "id": "claude-opus-4-7",
          "display_name": "Opus 4.7"
        },
        "effort": {
          "level": "high"
        },
        "workspace": {
          "current_dir": "@TMP@/code/app"
        },
        "context_window": {
          "used_percentage": 42.4,
          "total_input_tokens": 1500,
          "total_output_tokens": 23400,
          "current_usage": {
            "input_tokens": 2000,
            "cache_creation_input_tokens": 1000,
            "cache_read_input_tokens": 17000
          }
        },
        "rate_limits": {
          "five_hour": {
            "used_percentage": 45.4,
            "resets_at": 1760009000
          },
          "seven_day": {
            "used_percentage": 62,
            "resets_at": 1760302400
          }
        },
        "transcript_path": "@TMP@/transcript.jsonl"
      }
      """
    And the config:
      """
      theme: saber
      segments: [path, model, ctx, tokens, cache, rate5h, rate7d]
      """
    And the file "code/app/.git/HEAD":
      """
      ref: refs/heads/feature/retry-backoff
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- review: Review a change for correctness, style and missing tests before it is merged\u000a- debug: Work out why a test fails or a service misbehaves, starting from the evidence\u000a- plan: Write an implementation plan for a multi-step change before touching code\u000a- release: Prepare release notes and check the version label on a pull request\u000a- docs: Write or restructure a repository's README and contributing guide\u000a"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      """
    When the status line renders at 200 columns
    Then the output is saved as the screenshot "saber"

  @screenshot
  Scenario: The saber theme's rate-limit bar with NO_COLOR, showing it still reads without colour
    Given the session:
      """json
      {
        "session_id": "demo",
        "rate_limits": {
          "five_hour": {
            "used_percentage": 75,
            "resets_at": 1760009000
          }
        }
      }
      """
    And the config:
      """
      theme: saber
      segments: [rate5h]
      """
    And the environment variable "NO_COLOR" is "1"
    When the status line renders at 200 columns
    Then the output is saved as the screenshot "saber-no-color"

  @screenshot
  Scenario: The status line at 80 columns, wrapped into rows, for the customising guide
    Given the session:
      """json
      {
        "session_id": "demo",
        "model": {
          "id": "claude-opus-4-7",
          "display_name": "Opus 4.7"
        },
        "effort": {
          "level": "high"
        },
        "workspace": {
          "current_dir": "@TMP@/code/app"
        },
        "context_window": {
          "used_percentage": 42.4,
          "total_input_tokens": 1500,
          "total_output_tokens": 23400,
          "current_usage": {
            "input_tokens": 2000,
            "cache_creation_input_tokens": 1000,
            "cache_read_input_tokens": 17000
          }
        },
        "rate_limits": {
          "five_hour": {
            "used_percentage": 45.4,
            "resets_at": 1760009000
          },
          "seven_day": {
            "used_percentage": 62,
            "resets_at": 1760302400
          }
        },
        "transcript_path": "@TMP@/transcript.jsonl"
      }
      """
    And the file "code/app/.git/HEAD":
      """
      ref: refs/heads/feature/retry-backoff
      """
    And the file "transcript.jsonl":
      """
      {"type":"user","message":{"role":"user","content":"hi"}}
      {"type":"attachment","attachment":{"type":"skill_listing","isInitial":true,"content":"- review: Review a change for correctness, style and missing tests before it is merged\u000a- debug: Work out why a test fails or a service misbehaves, starting from the evidence\u000a- plan: Write an implementation plan for a multi-step change before touching code\u000a- release: Prepare release notes and check the version label on a pull request\u000a- docs: Write or restructure a repository's README and contributing guide\u000a"}}
      {"type":"assistant","message":{"model":"claude-opus-4-7","usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500,"output_tokens":5}}}
      """
    When the status line renders at 80 columns
    Then the output is saved as the screenshot "narrow"

  @screenshot
  Scenario: The status line with a custom segment script that prints a red backup warning, for the customising guide
    Given the session:
      """json
      {
        "session_id": "demo",
        "model": {
          "id": "claude-opus-4-7",
          "display_name": "Opus 4.7"
        },
        "effort": {
          "level": "high"
        },
        "workspace": {
          "current_dir": "@TMP@/code/app"
        },
        "context_window": {
          "used_percentage": 42.4,
          "total_input_tokens": 1500,
          "total_output_tokens": 23400,
          "current_usage": {
            "input_tokens": 2000,
            "cache_creation_input_tokens": 1000,
            "cache_read_input_tokens": 17000
          }
        },
        "rate_limits": {
          "five_hour": {
            "used_percentage": 45.4,
            "resets_at": 1760009000
          },
          "seven_day": {
            "used_percentage": 62,
            "resets_at": 1760302400
          }
        }
      }
      """
    And the config:
      """
      segments: [path, model, ctx, tokens, cache, rate5h, rate7d]
      """
    And the file "code/app/.git/HEAD":
      """
      ref: refs/heads/feature/retry-backoff
      """
    And the executable file "statusline.d/10-backup.sh":
      """
      #!/bin/sh
      # statusline-ttl: 300
      printf '\033[31m⚠ backup stale\033[0m'
      """
    When the status line renders at 200 columns
    Then the output is saved as the screenshot "custom-segment"

  @screenshot
  Scenario: The agent panel rows the subagent command prints, for the customising guide
    Given the session:
      """json
      {
        "columns": 200,
        "tasks": [
          {
            "id": "a",
            "name": "reviewer",
            "type": "local_agent",
            "description": "Review the payment retry change",
            "model": "claude-sonnet-4-5-20250929",
            "effort": "high",
            "contextWindowSize": 200000,
            "tokenCount": 84500
          },
          {
            "id": "b",
            "name": "tester",
            "type": "local_agent",
            "description": "Run the suite",
            "model": "claude-opus-4-7",
            "effort": "xhigh",
            "contextWindowSize": 200000,
            "tokenCount": 185000
          }
        ]
      }
      """
    When the subagent rows render
    Then the output is saved as the screenshot "subagent"
