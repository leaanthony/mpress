package translate

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/leaanthony/mpress/internal/config"
)

// ResolveAPIKey reads the configured environment variable first, then the
// private per-user provider file. Credentials are never copied into a project.
func ResolveAPIKey(cfg config.Config) string {
	return resolveProviderAPIKey(cfg.Translation.Provider, cfg.Translation.APIKeyEnv)
}

// DetectModelCapabilities performs read-only local detection for the model
// selection truth table. It does not execute a harness or contact a provider.
func DetectModelCapabilities(cfg config.Config) ModelCapabilities {
	codexCommand := "codex"
	claudeCommand := "claude"
	if command := strings.TrimSpace(cfg.Translation.Command); command != "" {
		switch strings.ToLower(strings.TrimSpace(cfg.Translation.Provider)) {
		case "codex":
			codexCommand = command
		case "claude":
			claudeCommand = command
		}
	}
	_, codexErr := exec.LookPath(codexCommand)
	_, claudeErr := exec.LookPath(claudeCommand)
	return ModelCapabilities{
		CodexSubscription:      codexErr == nil,
		ClaudeCodeSubscription: claudeErr == nil,
		OpenRouterAPI:          resolveProviderAPIKey("openrouter", "OPENROUTER_API_KEY") != "",
		OpenAIAPI:              resolveProviderAPIKey("openai", "OPENAI_API_KEY") != "",
	}
}

func resolveProviderAPIKey(provider, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" || strings.ContainsAny(provider, `/\`) {
		return ""
	}
	directory, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(directory, "mpress", provider+".env")
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return ""
	}
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(scanner.Text()), "export "))
		key, value, found := strings.Cut(line, "=")
		if !found || strings.TrimSpace(key) != name {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '\'' && value[len(value)-1] == '\'' || value[0] == '"' && value[len(value)-1] == '"') {
			value = value[1 : len(value)-1]
		}
		return strings.TrimSpace(value)
	}
	return ""
}
