@config_command
Feature: Config command
  The config command prints every setting with its default and its comment, with each line commented out, so the output is a starting point for your own config file.

  Scenario: The config command prints the defaults, commented out
    When the config command runs
    Then the text starts with:
      """
      # # claude-statusline settings, all commented out so the defaults still apply.
      # # Save your file as config.yaml in the claude-statusline folder of your Claude config directory.
      # # "claude-statusline help" shows the exact path.
      # # To change a setting, remove the "# " in front of it.
      # # When it sits in a section, remove the "# " in front of the section header too.
      # # Delete the settings you do not change, so that future defaults reach you.
      # # If you toggle comments in an editor, the explanations stay commented,
      # # because only the lines you pick lose their "# ".
      #
      # # Segments to render, in order.
      # segments: [path, model, ctx, tokens, cache, cost, boot, skills, skills_stripped, agent_skills, mem, rate5h, rate7d]
      """
