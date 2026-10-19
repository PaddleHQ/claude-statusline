@help
Feature: Help
  The help command describes the segments the current config shows, and what each hook adds.

  Scenario: With no config file and no hooks, the defaults are described and the config file is reported as not created
    When help runs
    Then the text is:
      """
      Your status line, as currently configured.

      Shown now, in order:
        path: The working directory, with your home directory shown as ~, followed by the git branch in brackets.
        model: The model in use and its effort level, as in Opus 4.7 [high]. It reads Haiku (skill), for example, while a skill or subagent runs a different model than the session's.
        ctx: How full the context window is. Amber from 50%, red from 90%.
        tokens: Session token totals: input after the down arrow, output after the up arrow.
        cache: How much of the latest request's input was read from the prompt cache. Red below 5%.
        boot: The prompt size on the first turn, before you said anything.
        skills: The size of the always-loaded skill listing, in tokens.
        skills_stripped: Red when skills lost their descriptions to fit the budget; shows how many.
        agent_skills: Amber when a subagent lost skill descriptions; shows how many.
        mem: Red when a SessionStart hook output was too large to inject.
        rate5h: Your 5 hour usage, with a marker at an even pace. Amber when more than 20 points off it, red beyond 40 points or from 95% used.
        rate7d: Your 7 day usage, with a marker at an even pace. Amber when more than 10 points off it, red beyond 20 points or from 95% used.

      Config file: @TMP@/claude-statusline/config.yaml (not created yet, so the defaults apply)
      Documentation: https://github.com/PaddleHQ/claude-statusline/blob/main/docs/customising.md
      """

  Scenario: A hook named in the segments list replaces the built-in of that name, and is described as the hook
    Given the config:
      """
      segments: [path, model, ctx]
      """
    And the executable file "statusline.d/model.sh":
      """
      #!/bin/sh
      # statusline-help: My own model label
      printf nothing
      """
    When help runs
    Then the text is:
      """
      Your status line, as currently configured.

      Shown now, in order:
        path: The working directory, with your home directory shown as ~, followed by the git branch in brackets.
        model: My own model label
        ctx: How full the context window is. Amber from 50%, red from 90%.

      Custom segments:
        model (model.sh, slot ctx): My own model label

      Also available, not in your segments list:
        tokens: Session token totals: input after the down arrow, output after the up arrow.
        cache: How much of the latest request's input was read from the prompt cache. Red below 5%.
        rate5h: Your 5 hour usage, with a marker at an even pace. Amber when more than 20 points off it, red beyond 40 points or from 95% used.
        rate7d: Your 7 day usage, with a marker at an even pace. Amber when more than 10 points off it, red beyond 20 points or from 95% used.
        boot: The prompt size on the first turn, before you said anything.
        skills: The size of the always-loaded skill listing, in tokens.
        skills_stripped: Red when skills lost their descriptions to fit the budget; shows how many.
        agent_skills: Amber when a subagent lost skill descriptions; shows how many.
        mem: Red when a SessionStart hook output was too large to inject.

      Config file: @TMP@/claude-statusline/config.yaml
      Documentation: https://github.com/PaddleHQ/claude-statusline/blob/main/docs/customising.md
      """
