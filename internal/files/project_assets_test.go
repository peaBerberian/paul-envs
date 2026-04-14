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
		GitName:           "Test User",
		GitEmail:          "test@example.com",
	}

	runtimeTplData := RuntimeTemplateData{
		Version:         "1.0.0",
		ProjectHostPath: "/host/path",
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
		`GIT_AUTHOR_NAME Test User`,
		`GIT_AUTHOR_EMAIL test@example.com`,
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
