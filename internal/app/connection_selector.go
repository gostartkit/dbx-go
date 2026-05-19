package app

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"pkg.gostartkit.com/dbx/internal/config"
	"pkg.gostartkit.com/dbx/internal/util"
)

func sortedConnections(connections []config.ConnectionConfig, lastUsed string) []config.ConnectionConfig {
	ordered := append([]config.ConnectionConfig(nil), connections...)
	sort.SliceStable(ordered, func(i, j int) bool {
		iLast := ordered[i].Name == lastUsed
		jLast := ordered[j].Name == lastUsed
		if iLast != jLast {
			return iLast
		}
		return ordered[i].Name < ordered[j].Name
	})
	return ordered
}

func formatConnectionSummary(connection config.ConnectionConfig) string {
	if connection.Mode == "proxy-ssh" && connection.Proxy != nil && connection.SSH != nil {
		return fmt.Sprintf("%-8s %-5s %-9s %s via %s -> %s", connection.Name, connection.Driver, connection.Mode, connection.Address(), config.RedactProxyURL(connection.Proxy.URL), connection.SSH.Host)
	}
	if connection.Mode == "proxy" && connection.Proxy != nil {
		return fmt.Sprintf("%-8s %-5s %-6s %s via %s", connection.Name, connection.Driver, connection.Mode, connection.Address(), config.RedactProxyURL(connection.Proxy.URL))
	}
	if connection.Mode == "ssh" && connection.SSH != nil {
		return fmt.Sprintf("%-8s %-5s %-6s %s via %s", connection.Name, connection.Driver, connection.Mode, connection.Address(), connection.SSH.Host)
	}
	return fmt.Sprintf("%-8s %-5s %-6s %s", connection.Name, connection.Driver, connection.Mode, connection.Address())
}

func (a *Application) promptForConnectionSelection(ctx context.Context, connections []config.ConnectionConfig) (string, error) {
	if len(connections) == 0 {
		return "", nil
	}

	lastUsed := ""
	sessionFile, err := a.store.LoadSession()
	if err == nil {
		lastUsed = sessionFile.CurrentConnection
	}
	ordered := sortedConnections(connections, lastUsed)

	for index, connection := range ordered {
		a.prompt.Printf("%d) %s\n", index+1, formatConnectionSummary(connection))
	}

	options := make([]string, 0, len(ordered))
	for _, connection := range ordered {
		options = append(options, connection.Name)
	}

	for {
		value, err := a.ask(ctx, "Select connection by number or name", "")
		if err != nil {
			return "", err
		}

		if index, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
			if index >= 1 && index <= len(ordered) {
				return ordered[index-1].Name, nil
			}
		}

		if slices.Contains(options, value) {
			return value, nil
		}

		a.prompt.Println(util.WrapLayer("validation", "select connection", fmt.Errorf("please choose a listed number or connection name")).Error())
	}
}
