package segment

type model struct{}

func (model) Name() string { return "model" }

func (model) Keys() []string { return []string{"model.effort"} }

func (model) Render(ctx *Context) string {
	name := ctx.Payload.Model.DisplayName
	if name == "" {
		return ""
	}
	if level := ctx.Payload.Effort.Level; level != "" && ctx.Config.Bool("model.effort") {
		name += " [" + level + "]"
	}
	return ctx.Theme.Paint(ctx.Theme.Text, name)
}
