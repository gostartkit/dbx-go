package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
)

type Service struct {
	store *config.Store
}

func NewService(store *config.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Resolve(command string, cfg *config.ConnectionConfig) (*Template, error) {
	templates, err := s.List(command, cfg)
	if err != nil {
		return nil, err
	}
	if len(templates) == 0 {
		return nil, fmt.Errorf("no template found for command %q and driver %q", command, cfg.Driver)
	}
	chosen := templates[0]
	return &chosen, nil
}

func (s *Service) List(command string, cfg *config.ConnectionConfig) ([]Template, error) {
	layers := []struct {
		layer     string
		sourceDir string
		builtins  []Template
	}{
		{
			layer:     "connection",
			sourceDir: s.store.ConnectionTemplatesDir(cfg.Name),
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

	matchesFound := make([]Template, 0)
	for _, layer := range layers {
		var templates []Template
		var err error
		if layer.sourceDir != "" {
			templates, err = s.loadDir(layer.sourceDir, layer.layer)
			if err != nil {
				return nil, fmt.Errorf("load %s templates: %w", layer.layer, err)
			}
		} else {
			templates = layer.builtins
		}

		for _, tpl := range templates {
			if matches(tpl, command, cfg.Driver) {
				matchesFound = append(matchesFound, tpl)
			}
		}
	}

	return matchesFound, nil
}

func (s *Service) ResolveNamed(command string, cfg *config.ConnectionConfig, name string) (*Template, error) {
	templates, err := s.List(command, cfg)
	if err != nil {
		return nil, err
	}

	for _, tpl := range templates {
		if strings.EqualFold(strings.TrimSpace(tpl.Name), strings.TrimSpace(name)) {
			chosen := tpl
			return &chosen, nil
		}
	}

	return nil, fmt.Errorf("template %q not found for command %q and driver %q", name, command, cfg.Driver)
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
