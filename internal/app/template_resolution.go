package app

import (
	"context"
	"fmt"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	tpl "pkg.gostartkit.com/dbx/internal/template"
)

func (a *Application) resolveTemplateForAction(ctx context.Context, command string, cfg *config.ConnectionConfig) (*tpl.Template, error) {
	match, err := a.templates.ResolveByLayer(command, cfg)
	if err != nil {
		return nil, err
	}
	if len(match.Templates) == 0 {
		return nil, fmt.Errorf("no template found for command %q and driver %q", command, templateDriver(cfg))
	}
	if len(match.Templates) == 1 {
		chosen := match.Templates[0]
		return &chosen, nil
	}
	return a.chooseTemplateFromLayer(ctx, command, match)
}

func (a *Application) chooseTemplateFromLayer(ctx context.Context, command string, match *tpl.LayerMatch) (*tpl.Template, error) {
	if match == nil || len(match.Templates) == 0 {
		return nil, fmt.Errorf("no template candidates available for command %q", command)
	}

	options := make([]string, 0, len(match.Templates))
	indexByOption := make(map[string]int, len(match.Templates))
	for idx, candidate := range match.Templates {
		option := templateCandidateOption(candidate)
		options = append(options, option)
		indexByOption[option] = idx
	}

	selected, err := a.choose(ctx, fmt.Sprintf("Template for %s", command), options, "")
	if err != nil {
		return nil, err
	}

	index, ok := indexByOption[selected]
	if !ok {
		return nil, fmt.Errorf("selected template option %q not found", selected)
	}

	chosen := match.Templates[index]
	return &chosen, nil
}

func templateCandidateOption(candidate tpl.Template) string {
	parts := []string{
		candidate.Name,
		fmt.Sprintf("scope=%s", candidate.Layer),
		fmt.Sprintf("category=%s", candidate.EffectiveCategory()),
	}
	if description := strings.TrimSpace(candidate.Description); description != "" {
		parts = append(parts, fmt.Sprintf("description=%s", description))
	}
	if source := strings.TrimSpace(candidate.Source); source != "" {
		parts = append(parts, fmt.Sprintf("source=%s", source))
	}
	return strings.Join(parts, " | ")
}

func buildCLITemplateAmbiguityError(command string, match *tpl.LayerMatch) error {
	if match == nil || len(match.Templates) == 0 {
		return fmt.Errorf("no template found for command %q", command)
	}

	lines := []string{
		fmt.Sprintf("multiple templates matched command %q at %s scope:", command, match.Layer),
	}
	for _, candidate := range match.Templates {
		lines = append(lines, "  - "+templateCandidateOption(candidate))
	}
	lines = append(lines, "choose one explicitly with run template <name> or --template <name>")
	return fmt.Errorf("%s", strings.Join(lines, "\n"))
}

func templateDriver(cfg *config.ConnectionConfig) string {
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.Driver)
}
