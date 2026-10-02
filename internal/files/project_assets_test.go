package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	versions "github.com/peaberberian/paul-envs/internal"
)

func TestFileStore_CreateProjectFiles(t *testing.T) {
	baseDataDir := t.TempDir()
	configDir := t.TempDir()
	store := &FileStore{
		userFS: &UserFS{
			homeDir:  t.TempDir(),
			sudoUser: nil,
		},
		baseDataDir:   baseDataDir,
		baseConfigDir: configDir,
		projectsDir:   filepath.Join(baseDataDir, "projects"),
	}

	buildTplData := BuildTemplateData{
		Version:           "1.0.0",
		HostUID:           "1000",
		HostGID:           "1000",
		Username:          "testuser",
		Shell:             "bash",
		InstallNode:       "latest",
		InstallRust:       "none",
		InstallPython:     "3.12.0",
		InstallGo:         "none",
		EnableWasm:        "false",
		EnableSSH:         "true",
		EnableSudo:        "true",
		Packages:          "git vim",
		InstallNeovim:     "true",
		InstallStarship:   "true",
		InstallOhMyPosh:   "true",
		InstallAtuin:      "false",
		InstallMise:       "true",
		InstallZellij:     "false",
		InstallJujutsu:    "false",
		InstallDelta:      "false",
		InstallOpenCode:   "false",
		InstallClaudeCode: "false",
		InstallCodex:      "false",
		InstallFirefox:    "false",
	}

	runtimeTplData := RuntimeTemplateData{
		Version:         "1.0.0",
		ProjectHostPath: "/host/path",
		DotfilesPath:    "dotfiles",
		GitName:         "Test User",
		GitEmail:        "test@example.com",
		Volumes:         []string{"./vol1:/data1:ro", "./vol2:/data2"},
		Ports:           []string{"3000:3000", "8080:8080"},
	}

	err := store.CreateProjectFiles("testproject", buildTplData, runtimeTplData)
	if err != nil {
		t.Fatalf("CreateProjectFiles() error = %v", err)
	}

	buildFile := store.GetProjectBuildConfigPath("testproject")
	if _, err := os.Stat(buildFile); os.IsNotExist(err) {
		t.Fatal("build.conf was not created")
	}
	runFile := store.GetProjectRuntimeConfigPath("testproject")
	if _, err := os.Stat(runFile); os.IsNotExist(err) {
		t.Fatal("run.conf was not created")
	}
	projectInfoFile := store.getProjectInfoFilePathFor("testproject")
	if _, err := os.Stat(projectInfoFile); os.IsNotExist(err) {
		t.Fatal("project.lock file was not created")
	}
	dotfilesDir := store.GetProjectDotfilesPath("testproject")
	if info, err := os.Stat(dotfilesDir); err != nil || !info.IsDir() {
		t.Fatalf("project dotfiles directory was not created: stat err=%v", err)
	}
	if filepath.Dir(projectInfoFile) != store.getProjectInternalDir("testproject") {
		t.Fatalf("project.lock should be stored in %s, got %s", store.getProjectInternalDir("testproject"), filepath.Dir(projectInfoFile))
	}

	buildCtnt, err := os.ReadFile(buildFile)
	if err != nil {
		t.Fatal(err)
	}
	buildChecks := []string{
		`VERSION 1.0.0`,
		`USERNAME testuser`,
		`USER_SHELL bash`,
		`INSTALL_NODE latest`,
		`ENABLE_SSH true`,
		`HOST_UID 1000`,
	}
	for _, check := range buildChecks {
		if !strings.Contains(string(buildCtnt), check) {
			t.Errorf("build.conf missing expected content: %s", check)
		}
	}

	runCtnt, err := os.ReadFile(runFile)
	if err != nil {
		t.Fatal(err)
	}
	runChecks := []string{
		`VERSION 1.0.0`,
		`PATH /host/path`,
		`DOTFILES_PATH dotfiles`,
		`GIT_AUTHOR_NAME Test User`,
		`GIT_AUTHOR_EMAIL test@example.com`,
		`VOLUME ./vol1:/data1:ro`,
		`VOLUME ./vol2:/data2`,
		`PORT 3000:3000`,
		`PORT 8080:8080`,
	}
	for _, check := range runChecks {
		if !strings.Contains(string(runCtnt), check) {
			t.Errorf("run.conf missing expected content: %s", check)
		}
	}

	pInfoCtnt, err := os.ReadFile(projectInfoFile)
	if err != nil {
		t.Fatal(err)
	}
	pInfoChecks := []string{
		`VERSION=` + versions.ProjectLockVersion.ToString(),
		`DOCKERFILE_VERSION=` + versions.DockerfileVersion.ToString(),
	}
	for _, check := range pInfoChecks {
		if !strings.Contains(string(pInfoCtnt), check) {
			t.Errorf("project.lock file missing expected content: %s", check)
		}
	}
}

func TestEmbeddedDockerfileCodexInstaller(t *testing.T) {
	dockerfile, err := assets.ReadFile("embeds/Dockerfile")
	if err != nil {
		t.Fatal(err)
	}

	content := string(dockerfile)
	header := "# Dockerfile - Version: " + versions.DockerfileVersion.ToString()
	if !strings.Contains(content, header) {
		t.Errorf("embedded Dockerfile missing version header %q", header)
	}
	if !strings.Contains(content, "curl -fsSL https://chatgpt.com/codex/install.sh | sh") {
		t.Error("embedded Dockerfile does not use the Codex standalone installer")
	}
	if !strings.Contains(content, "export CODEX_HOME=/home/${USERNAME}/.local/share/codex-cli") {
		t.Error("embedded Dockerfile does not keep the Codex CLI package outside persistent runtime state")
	}
	if strings.Contains(content, "github.com/openai/codex/releases/download") {
		t.Error("embedded Dockerfile still downloads the incomplete Codex release binary")
	}
}

func TestEmbeddedDockerfileCurrentToolInstallers(t *testing.T) {
	dockerfile, err := assets.ReadFile("embeds/Dockerfile")
	if err != nil {
		t.Fatal(err)
	}

	content := string(dockerfile)
	expected := []string{
		"curl --proto '=https' --tlsv1.2 -LsSf https://setup.atuin.sh | sh",
		"curl -fsSL https://mise.run | sh",
		"https://github.com/jj-vcs/jj/releases/download/",
	}
	for _, value := range expected {
		if !strings.Contains(content, value) {
			t.Errorf("embedded Dockerfile missing current installer %q", value)
		}
	}

	obsolete := []string{
		"https://mise.jdx.dev/install.sh",
		"https://github.com/martinvonz/jj/releases/download/",
	}
	for _, value := range obsolete {
		if strings.Contains(content, value) {
			t.Errorf("embedded Dockerfile still uses obsolete installer %q", value)
		}
	}
}
