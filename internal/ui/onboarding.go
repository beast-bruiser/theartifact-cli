package ui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
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

// PrintNoWorkspacesHint prints a friendly message and the URL to create a workspace.
func PrintNoWorkspacesHint() {
	fmt.Println()
	PrintWarn("No workspaces found.")
	fmt.Println(MutedStyle.Render("  Create one at: ") + HintStyle.Render("https://theartifact.art/workspaces"))
	fmt.Println()
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
