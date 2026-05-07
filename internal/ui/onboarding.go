package ui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"
)

const APIKeysURL = "https://theartifact.art/api-keys"

// PromptAPIKey runs the interactive API key onboarding flow.
// It prints the API keys URL, offers to open the browser (TTY only),
// then reads and validates the pasted key.
// Returns the validated key or an error if the user cancels.
func PromptAPIKey(scanner *bufio.Scanner) (string, error) {
	fmt.Println()
	fmt.Println(AccentStyle.Render("  ● Welcome to TheArtifact"))
	fmt.Println(MutedStyle.Render("  AI-powered asset generation · theartifact.art"))
	fmt.Println()
	fmt.Println(MutedStyle.Render("  You need an API key to continue."))
	fmt.Printf("  %s\n", HintStyle.Render(APIKeysURL))
	fmt.Println()

	if isTTY() {
		fmt.Print(MutedStyle.Render("  [Enter] open in browser · or paste your key now: "))
	} else {
		fmt.Print(AccentStyle.Render("  API key ▸ "))
	}

	for {
		if !scanner.Scan() {
			fmt.Println()
			return "", fmt.Errorf("authentication cancelled")
		}
		key := strings.TrimSpace(scanner.Text())

		// Empty input on a TTY = open browser
		if key == "" && isTTY() {
			openBrowser(APIKeysURL)
			fmt.Println()
			fmt.Print(AccentStyle.Render("  API key ▸ "))
			continue
		}

		if err := validateAPIKey(key); err != nil {
			PrintWarn(err.Error())
			fmt.Print(AccentStyle.Render("  API key ▸ "))
			continue
		}

		return key, nil
	}
}

// PromptWorkspace shows a numbered workspace picker and returns the selected id and name.
// Returns empty strings if the user provides an invalid selection.
func PromptWorkspace(scanner *bufio.Scanner, workspaces []WorkspaceItem) (id, name string) {
	fmt.Println()
	for i, w := range workspaces {
		fmt.Printf("  %s  %s\n",
			AccentStyle.Render(fmt.Sprintf("[%d]", i+1)),
			BoldStyle.Render(w.Name)+MutedStyle.Render("  "+w.ID),
		)
	}
	fmt.Println()
	fmt.Print(AccentStyle.Render("  workspace ▸ "))

	if !scanner.Scan() {
		return "", ""
	}
	choice := strings.TrimSpace(scanner.Text())

	for i, w := range workspaces {
		if choice == fmt.Sprintf("%d", i+1) || strings.EqualFold(choice, w.ID) {
			return w.ID, w.Name
		}
	}
	return "", ""
}

// DeriveWorkspaceName creates a workspace name from the folder path.
// Non-alphanumeric characters (except space) become dashes; falls back to "my-workspace".
func DeriveWorkspaceName(cwd string) string {
	base := filepath.Base(cwd)
	var b strings.Builder
	for _, r := range base {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	name := strings.Trim(b.String(), "- ")
	if name == "" {
		return "my-workspace"
	}
	return name
}

// PromptWorkspaceName shows the suggested name and lets the user confirm or edit it.
func PromptWorkspaceName(scanner *bufio.Scanner, suggested string) (string, error) {
	fmt.Println()
	fmt.Printf("  %s %s  %s\n",
		AccentStyle.Render("Workspace name:"),
		BoldStyle.Render(suggested),
		MutedStyle.Render("[Enter to keep, or type a new name]"),
	)
	fmt.Print(AccentStyle.Render("  name ▸ "))

	if !scanner.Scan() {
		return "", fmt.Errorf("init cancelled")
	}
	if name := strings.TrimSpace(scanner.Text()); name != "" {
		return name, nil
	}
	return suggested, nil
}

// PromptCreateOrLink asks whether to create a new workspace or link an existing one.
// Returns true if the user chooses to create a new workspace.
func PromptCreateOrLink(scanner *bufio.Scanner, folderName string) bool {
	fmt.Println()
	fmt.Println(MutedStyle.Render("  You have existing workspaces."))
	fmt.Printf("  %s  %s\n", AccentStyle.Render("[1]"), BoldStyle.Render(fmt.Sprintf("Create new workspace \"%s\"", folderName)))
	fmt.Printf("  %s  %s\n", AccentStyle.Render("[2]"), BoldStyle.Render("Link an existing workspace"))
	fmt.Println()
	fmt.Print(AccentStyle.Render("  choice ▸ "))

	if !scanner.Scan() {
		return true
	}
	return strings.TrimSpace(scanner.Text()) != "2"
}

// WorkspaceItem is a minimal workspace representation for the picker.
type WorkspaceItem struct {
	ID   string
	Name string
}

// ── helpers ───────────────────────────────────────────────────────────────────

func validateAPIKey(key string) error {
	if key == "" {
		return fmt.Errorf("Key cannot be empty. Try again.")
	}
	if !strings.HasPrefix(key, "tak_live_") && !strings.HasPrefix(key, "tak_test_") {
		return fmt.Errorf("That doesn't look like a valid key (expected tak_live_... or tak_test_...). Try again.")
	}
	if utf8.RuneCountInString(key) < 20 {
		return fmt.Errorf("Key looks too short. Double-check and try again.")
	}
	return nil
}

func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
	PrintInfo("Opening browser…")
}
