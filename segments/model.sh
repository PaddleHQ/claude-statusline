segment_model() {
  [ -n "$MODEL_NAME" ] || return 0
  model_text=$MODEL_NAME
  config_bool model.effort
  if [ -n "$EFFORT" ] && [ "$CFG_VALUE" = true ]; then
    model_text="$model_text [$EFFORT]"
  fi
  paint "$C_TEXT" "$model_text"
  printf '%s' "$PAINTED"
}
