package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	colorful "github.com/lucasb-eyer/go-colorful"
)

var (
	substitutionRE = regexp.MustCompile(`\${((\\.|.)+?)}`)
)

// SmartButtonWidget is a button widget that can change dynamically.
type SmartButtonWidget struct {
	*ButtonWidget

	iconTemplate     string
	labelTemplate    string
	fontsizeTemplate string
	colorTemplate    string
	currentIcon      string
	dependencies     []SmartButtonDependency
}

// SmartButtonDependency is some dependency of the smart button.
type SmartButtonDependency interface {
	IsNecessary(expressions []string) bool
	IsChanged() bool
	Replacement(expression string) (string, bool)
}

// SmartButtonDependencyBase is the base structure of a dependency.
type SmartButtonDependencyBase struct {
	toBeReplaced []string
}

// NewSmartButtonDependencyBase returns a new SmartButtonDependencyBase.
func NewSmartButtonDependencyBase(toBeReplaced ...string) *SmartButtonDependencyBase {
	return &SmartButtonDependencyBase{
		toBeReplaced: toBeReplaced,
	}
}

// IsNecessary returns whether the dependency is necessary for the template.
func (d *SmartButtonDependencyBase) IsNecessary(expressions []string) bool {
	for _, x := range expressions {
		for _, r := range d.toBeReplaced {
			if x == r {
				return true
			}
		}
	}
	return false
}

// IsChanged returns true if the dependency value has changed.
func (d *SmartButtonDependencyBase) IsChanged() bool {
	return false
}

// Replacement returns the replacement value for expression and whether it applies.
func (d *SmartButtonDependencyBase) Replacement(expression string) (string, bool) {
	return expression, false
}

// SmartButtonBrightnessDependency is a dependency based on the brightness setting.
type SmartButtonBrightnessDependency struct {
	*SmartButtonDependencyBase

	brightness uint
}

// NewSmartButtonBrightnessDependency returns a new SmartButtonBrightnessDependency.
func NewSmartButtonBrightnessDependency() *SmartButtonBrightnessDependency {
	return &SmartButtonBrightnessDependency{
		SmartButtonDependencyBase: NewSmartButtonDependencyBase("brightness"),
		brightness:                *brightness,
	}
}

// IsChanged returns true if the brightness has changed.
func (d *SmartButtonBrightnessDependency) IsChanged() bool {
	return d.brightness != *brightness
}

// Replacement returns the replacement value for expression and whether it applies.
func (d *SmartButtonBrightnessDependency) Replacement(
	expression string,
) (string, bool) {
	if expression == d.toBeReplaced[0] {
		d.brightness = *brightness
		return fmt.Sprintf("%d", d.brightness), true
	}
	return expression, false
}

// NewSmartButtonWidget returns a new SmartButtonWidget.
func NewSmartButtonWidget(bw *BaseWidget, opts WidgetConfig) (*SmartButtonWidget, error) {
	var icon, label, fontsize, color string
	_ = ConfigValue(opts.Config["icon"], &icon)
	_ = ConfigValue(opts.Config["label"], &label)
	_ = ConfigValue(opts.Config["fontsize"], &fontsize)
	_ = ConfigValue(opts.Config["color"], &color)

	opts.Config["icon"] = ""
	opts.Config["label"] = ""
	opts.Config["fontsize"] = 0
	opts.Config["color"] = ""
	parent, err := NewButtonWidget(bw, opts)
	if err != nil {
		return nil, err
	}

	w := SmartButtonWidget{
		ButtonWidget:     parent,
		iconTemplate:     icon,
		labelTemplate:    label,
		fontsizeTemplate: fontsize,
		colorTemplate:    color,
	}

	all := []string{icon, label, fontsize, color}
	count := 0
	for _, s := range all {
		count += strings.Count(s, "${")
	}
	expressions := make([]string, 0, count)
	for _, s := range all {
		for _, arr := range substitutionRE.FindAllStringSubmatch(s, -1) {
			expressions = append(
				expressions,
				arr[1],
			)
		}
	}
	w.appendDependencyIfNecessary(NewSmartButtonBrightnessDependency(), expressions)

	return &w, nil
}

// appendDependency appends the dependency if the label requires it.
func (w *SmartButtonWidget) appendDependencyIfNecessary(
	d SmartButtonDependency,
	expressions []string,
) {
	if d.IsNecessary(expressions) {
		w.dependencies = append(w.dependencies, d)
	}
}

// RequiresUpdate returns true when the widget wants to be repainted.
func (w *SmartButtonWidget) RequiresUpdate() bool {
	changed := false
	for _, d := range w.dependencies {
		changed = d.IsChanged() || changed
	}
	return changed || w.ButtonWidget.RequiresUpdate()
}

func (w *SmartButtonWidget) replaceValues(template string) string {
	return substitutionRE.ReplaceAllStringFunc(
		template,
		func(found string) string {
			match := substitutionRE.FindStringSubmatch(found)
			for _, d := range w.dependencies {
				if r, applies := d.Replacement(match[1]); applies {
					return r
				}
			}
			// nothing applied, return the expression back
			return found
		},
	)
}

// Update renders the widget.
func (w *SmartButtonWidget) Update() error {
	label := w.replaceValues(w.labelTemplate)
	icon := w.replaceValues(w.iconTemplate)
	fontsizeString := w.replaceValues(w.fontsizeTemplate)
	colorString := w.replaceValues(w.colorTemplate)

	w.label = label
	if icon != w.currentIcon {
		if icon == "" {
			w.SetImage(nil)
			w.currentIcon = icon
		} else if err := w.LoadImage(icon); err == nil {
			w.currentIcon = icon
		}
	}
	if fontsizeString == "" {
		w.fontsize = 0
	} else {
		if fontsize, err := strconv.ParseFloat(fontsizeString, 64); err == nil {
			w.fontsize = fontsize
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	if colorString == "" {
		w.color = DefaultColor
	} else {
		if color, err := colorful.Hex(colorString); err == nil {
			w.color = color
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	return w.ButtonWidget.Update()
}
