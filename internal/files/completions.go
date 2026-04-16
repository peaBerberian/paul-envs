package files

import "fmt"

func GetCompletionScript(shell string) (string, error) {
	var path string
	switch shell {
	case "bash":
		path = "embeds/completions/bash_completion.sh"
	case "zsh":
		path = "embeds/completions/zsh_completion.sh"
	case "fish":
		path = "embeds/completions/fish_completion.fish"
	default:
		return "", fmt.Errorf("invalid shell %q. Must be one of: bash, zsh, fish", shell)
	}

	content, err := assets.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s completion asset: %w", shell, err)
	}
	return string(content), nil
}
