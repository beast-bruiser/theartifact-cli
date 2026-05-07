package cmd

import (
	"bufio"
	"fmt"

	"theartifact-cli/internal/api"
	"theartifact-cli/internal/config"
	"theartifact-cli/internal/ui"
)

type workspaceGateOptions struct {
	Name     string // pre-set workspace name; skips name prompt
	LinkID   string // bind to an existing workspace ID directly
	NoPrompt bool   // non-interactive; requires Name or LinkID
}

// runWorkspaceGate resolves or creates a workspace for the project.
// Returns nil projCfg when the user aborts interactively.
func runWorkspaceGate(scanner *bufio.Scanner, client *api.Client, projCfg *config.ProjectConfig, cwd string, opts workspaceGateOptions) (*config.ProjectConfig, error) {
	if opts.NoPrompt && opts.Name == "" && opts.LinkID == "" {
		return nil, fmt.Errorf("--no-prompt requires --name <name> or --link <id>")
	}

	// --link: bind directly without creating
	if opts.LinkID != "" {
		projCfg.WorkspaceID = opts.LinkID
		ui.PrintSuccess("Linked to workspace: " + opts.LinkID)
		return projCfg, nil
	}

	fmt.Println()
	fmt.Println(ui.WarnStyle.Render("  No workspace linked to this project. Let's set one up."))

	sp := ui.NewSpinner("Checking existing workspaces…")
	sp.Start()
	workspaces, err := client.ListWorkspaces()
	if err != nil {
		sp.Fail("Could not fetch workspaces")
		return nil, err
	}
	sp.Stop("")

	createNew := true
	if len(workspaces) > 0 && !opts.NoPrompt {
		createNew = ui.PromptCreateOrLink(scanner, ui.DeriveWorkspaceName(cwd))
	}

	if createNew {
		name := opts.Name
		if name == "" {
			suggested := ui.DeriveWorkspaceName(cwd)
			name, err = ui.PromptWorkspaceName(scanner, suggested)
			if err != nil {
				return nil, err
			}
		}
		sp2 := ui.NewSpinner("Creating workspace…")
		sp2.Start()
		ws, err := client.CreateWorkspace(name)
		if err != nil {
			sp2.Fail("Failed to create workspace")
			return nil, err
		}
		sp2.Stop(fmt.Sprintf("Workspace \"%s\" created", ws.Name))
		projCfg.WorkspaceID = ws.ID
		projCfg.WorkspaceName = ws.Name
		return projCfg, nil
	}

	// Link existing: show picker
	items := make([]ui.WorkspaceItem, len(workspaces))
	for i, w := range workspaces {
		items[i] = ui.WorkspaceItem{ID: w.ID, Name: w.Name}
	}
	if len(workspaces) == 1 {
		projCfg.WorkspaceID = workspaces[0].ID
		projCfg.WorkspaceName = workspaces[0].Name
		ui.PrintSuccess(fmt.Sprintf("Auto-selected workspace: %s", workspaces[0].Name))
		return projCfg, nil
	}
	id, wsName := ui.PromptWorkspace(scanner, items)
	if id == "" {
		ui.PrintWarn("Invalid selection.")
		return nil, nil
	}
	projCfg.WorkspaceID = id
	projCfg.WorkspaceName = wsName
	ui.PrintSuccess(fmt.Sprintf("Workspace set to: %s", wsName))
	return projCfg, nil
}
