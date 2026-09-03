package templ

import "github.com/almerlucke/genny/template"

func NewTrigger(amp any, msg template.Template) *Templ {
	return New(template.Template{
		"command":   "trigger",
		"amplitude": amp,
		"tick":      template.NewParameter("tick", nil),
		"message":   msg,
	})
}

func NewDurationTrigger(duration any, amp any, msg template.Template) *Templ {
	return New(template.Template{
		"command":   "trigger",
		"duration":  duration,
		"amplitude": amp,
		"tick":      template.NewParameter("tick", nil),
		"message":   msg,
	})
}
