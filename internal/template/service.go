package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
)

type Service struct {
	store *config.Store
}

type LayerMatch struct {
	Command   string
	Driver    string
	Layer     string
	Templates []Template
}

type AmbiguousResolveError struct {
	Command    string
	Driver     string
	Layer      string
	Candidates []Template
}

func (e *AmbiguousResolveError) Error() string {
	if e == nil {
		return "ambiguous template match"
	}
	names := make([]string, 0, len(e.Candidates))
	for _, candidate := range e.Candidates {
		names = append(names, candidate.Name)
	}
	sort.Strings(names)
	return fmt.Sprintf(
		"multiple templates matched command %q at %s scope: %s",
		e.Command,
		e.Layer,
		strings.Join(names, ", "),
	)
}

func NewService(store *config.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Resolve(command string, cfg *config.ConnectionConfig) (*Template, error) {
	match, err := s.ResolveByLayer(command, cfg)
	if err != nil {
		return nil, err
	}
	if len(match.Templates) == 0 {
		return nil, fmt.Errorf("no template found for command %q and driver %q", command, driverName(cfg))
	}
	if len(match.Templates) > 1 {
		return nil, &AmbiguousResolveError{
			Command:    command,
			Driver:     match.Driver,
			Layer:      match.Layer,
			Candidates: append([]Template(nil), match.Templates...),
		}
	}
	chosen := match.Templates[0]
	return &chosen, nil
}

func (s *Service) ResolveByLayer(command string, cfg *config.ConnectionConfig) (*LayerMatch, error) {
	for _, layer := range s.layers(cfg) {
		templates, err := s.loadLayer(layer)
		if err != nil {
			return nil, err
		}

		matchesFound := make([]Template, 0)
		for _, candidate := range templates {
			if matches(candidate, command, driverName(cfg)) {
				matchesFound = append(matchesFound, candidate)
			}
		}
		if len(matchesFound) == 0 {
			continue
		}

		return &LayerMatch{
			Command:   command,
			Driver:    driverName(cfg),
			Layer:     layer.layer,
			Templates: matchesFound,
		}, nil
	}

	return &LayerMatch{
		Command: command,
		Driver:  driverName(cfg),
	}, nil
}

func (s *Service) ListResolved(cfg *config.ConnectionConfig) ([]Template, error) {
	templates, err := s.listAll(cfg)
	if err != nil {
		return nil, err
	}

	resolved := make([]Template, 0, len(templates))
	seen := make(map[string]struct{}, len(templates))
	for _, tpl := range templates {
		key := strings.ToLower(strings.TrimSpace(tpl.Name))
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		resolved = append(resolved, tpl)
	}

	slices.SortFunc(resolved, func(a, b Template) int {
		return strings.Compare(a.Name, b.Name)
	})
	return resolved, nil
}

func (s *Service) List(command string, cfg *config.ConnectionConfig) ([]Template, error) {
	templates, err := s.listAll(cfg)
	if err != nil {
		return nil, err
	}

	matchesFound := make([]Template, 0)
	for _, tpl := range templates {
		if matches(tpl, command, driverName(cfg)) {
			matchesFound = append(matchesFound, tpl)
		}
	}
	return matchesFound, nil
}

func (s *Service) ResolveNamed(command string, cfg *config.ConnectionConfig, name string) (*Template, error) {
	templates, err := s.List(command, cfg)
	if err != nil {
		return nil, err
	}

	target := strings.ToLower(strings.TrimSpace(name))
	if target == "" {
		return nil, fmt.Errorf("template name is required")
	}

	matchesFound := make([]Template, 0)
	bestRank := 99
	for _, tpl := range templates {
		if strings.ToLower(strings.TrimSpace(tpl.Name)) != target {
			continue
		}
		rank := templateLayerRank(tpl.Layer)
		if rank < bestRank {
			bestRank = rank
			matchesFound = []Template{tpl}
			continue
		}
		if rank == bestRank {
			matchesFound = append(matchesFound, tpl)
		}
	}

	if len(matchesFound) == 0 {
		return nil, fmt.Errorf("template %q not found for command %q and driver %q", name, command, driverName(cfg))
	}
	if len(matchesFound) > 1 {
		return nil, fmt.Errorf("multiple templates named %q found for command %q at %s scope", name, command, matchesFound[0].Layer)
	}

	chosen := matchesFound[0]
	return &chosen, nil
}

func (s *Service) ResolveNamedAny(cfg *config.ConnectionConfig, name string) (*Template, error) {
	templates, err := s.listAll(cfg)
	if err != nil {
		return nil, err
	}

	target := strings.ToLower(strings.TrimSpace(name))
	if target == "" {
		return nil, fmt.Errorf("template name is required")
	}

	matchesFound := make([]Template, 0)
	bestRank := 99
	for _, tpl := range templates {
		if strings.ToLower(strings.TrimSpace(tpl.Name)) != target {
			continue
		}
		rank := templateLayerRank(tpl.Layer)
		if rank < bestRank {
			bestRank = rank
			matchesFound = []Template{tpl}
			continue
		}
		if rank == bestRank {
			matchesFound = append(matchesFound, tpl)
		}
	}

	if len(matchesFound) == 0 {
		return nil, fmt.Errorf("template %q not found", name)
	}
	if len(matchesFound) > 1 {
		return nil, fmt.Errorf("multiple templates named %q found at %s scope", name, matchesFound[0].Layer)
	}

	chosen := matchesFound[0]
	return &chosen, nil
}

func (s *Service) listAll(cfg *config.ConnectionConfig) ([]Template, error) {
	all := make([]Template, 0)
	for _, layer := range s.layers(cfg) {
		templates, err := s.loadLayer(layer)
		if err != nil {
			return nil, err
		}
		all = append(all, templates...)
	}
	return all, nil
}

func (s *Service) connectionTemplatesDir(cfg *config.ConnectionConfig) string {
	if cfg == nil || strings.TrimSpace(cfg.Name) == "" {
		return ""
	}
	return s.store.ConnectionTemplatesDir(cfg.Name)
}

func templateLayerRank(layer string) int {
	switch layer {
	case "connection":
		return 0
	case "global":
		return 1
	default:
		return 2
	}
}

type templateLayer struct {
	layer     string
	sourceDir string
	builtins  []Template
}

func (s *Service) layers(cfg *config.ConnectionConfig) []templateLayer {
	return []templateLayer{
		{
			layer:     "connection",
			sourceDir: s.connectionTemplatesDir(cfg),
		},
		{
			layer:     "global",
			sourceDir: s.store.GlobalTemplatesDir(),
		},
		{
			layer:    "builtin",
			builtins: Builtins(),
		},
	}
}

func (s *Service) loadLayer(layer templateLayer) ([]Template, error) {
	if layer.sourceDir != "" {
		templates, err := s.loadDir(layer.sourceDir, layer.layer)
		if err != nil {
			return nil, fmt.Errorf("load %s templates: %w", layer.layer, err)
		}
		return templates, nil
	}
	return append([]Template(nil), layer.builtins...), nil
}

func driverName(cfg *config.ConnectionConfig) string {
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.Driver)
}

func (s *Service) loadDir(dir string, layer string) ([]Template, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	templates := make([]Template, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		var tpl Template
		if err := json.Unmarshal(data, &tpl); err != nil {
			return nil, fmt.Errorf("load template %s: %w", path, err)
		}
		if err := tpl.Validate(); err != nil {
			return nil, fmt.Errorf("load template %s: %w", path, err)
		}

		tpl.Layer = layer
		tpl.Source = path
		templates = append(templates, tpl)
	}

	slices.SortFunc(templates, func(a, b Template) int {
		return strings.Compare(a.Name, b.Name)
	})
	return templates, nil
}

func matches(tpl Template, command string, driver string) bool {
	if !strings.EqualFold(strings.TrimSpace(tpl.Match.Command), strings.TrimSpace(command)) {
		return false
	}
	if tpl.Match.Driver == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(tpl.Match.Driver), strings.TrimSpace(driver))
}
