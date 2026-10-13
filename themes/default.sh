# Basic ANSI colours and a bold default-foreground marker read well on light and dark terminals.
# shellcheck disable=SC2034 # the renderer reads these theme variables
theme_default() {
  ESC=$(printf '\033')
  C_TEXT=''
  C_MUTED="${ESC}[2m"
  C_GOOD="${ESC}[32m"
  C_WARN="${ESC}[33m"
  C_BAD="${ESC}[31m"
  C_MARKER="${ESC}[1m"
  RESET="${ESC}[0m"
  G_FULL='█'
  G_PART_1='▏'
  G_PART_2='▎'
  G_PART_3='▍'
  G_PART_4='▌'
  G_PART_5='▋'
  G_PART_6='▊'
  G_PART_7='▉'
  G_EMPTY='░'
  G_MARKER='┃'
}
