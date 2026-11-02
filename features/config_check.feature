@config_check
Feature: Checking the config file
  The config-check command says which config file is read and whether it can be used. When it cannot, it shows
  the line the parser reached, why, and the lines around it. It also lists the settings that nothing reads.

  Scenario: A config that is not valid YAML is reported with the line, the reason and the lines around it
    Given the config:
      """
      hooks:
        ttl: 60
       timeout: 2
      """
    When the config check runs
    Then the text is:
      """
      Config file: ~/claude-statusline/config.yaml
      2 did not find expected key

        1 | hooks:
      > 2 |   ttl: 60
        3 |  timeout: 2
        4 |

      The line is where the parser noticed the problem; the mistake may be just before it.
      """

  Scenario: A valid config is reported as parsed without errors
    Given the config:
      """
      segments: [path, model]
      hooks:
        ttl: 30
      """
    When the config check runs
    Then the text is:
      """
      Config file: ~/claude-statusline/config.yaml
      Parsed without errors.
      """

  Scenario: A setting uncommented without its section header is listed as unknown
    Given the config:
      """
      # hooks:
      ttl: 60
      """
    When the config check runs
    Then the text is:
      """
      Config file: ~/claude-statusline/config.yaml
      Parsed without errors.

      Unknown settings (ignored):
        ttl
      """

  Scenario: Plugin options in settings.json are reported with where each setting came from
    Given the file "settings.json":
      """
      {"enabledPlugins": {"statusline@claude-statusline": true},
       "pluginConfigs": {"statusline@claude-statusline": {"options": {"theme": "saber"}}}}
      """
    And the config:
      """
      theme: gradient
      separator: "|"
      """
    When the config check runs
    Then the text is:
      """
      Config file: ~/claude-statusline/config.yaml
      Parsed without errors.

      Plugin options: ~/settings.json (statusline@claude-statusline)
        theme: plugin options
        segments: default
        separator: config.yaml
        path.strip: default
        path.worktrees: default
        path.branch: default
        path.emoji: default
        model.effort: default
        snapshot.enabled: default
        snapshot.path: default
      """
