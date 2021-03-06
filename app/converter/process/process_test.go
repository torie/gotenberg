package process

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/thecodingmachine/gotenberg/app/config"
	gfile "github.com/thecodingmachine/gotenberg/app/converter/file"
)

func makeFile(workingDir string, fileName string) *gfile.File {
	filePath := fmt.Sprintf("%s%s", "../../../_tests/", fileName)
	absPath, _ := filepath.Abs(filePath)

	r, _ := os.Open(absPath)
	defer r.Close()

	f, _ := gfile.NewFile(workingDir, r, fileName)

	return f
}

func loadCommandConfigs(configurationFilePath string) {
	path, _ := filepath.Abs(configurationFilePath)
	c, _ := config.NewAppConfig(path)
	Load(c.CommandsConfig)
}

func TestLoad(t *testing.T) {
	path, _ := filepath.Abs("../../../_tests/configurations/gotenberg.yml")
	c, _ := config.NewAppConfig(path)
	Load(c.CommandsConfig)

	if c.CommandsConfig != forest.commandsConfig {
		t.Error("Commands configuration should have been loaded correctly")
	}
}

func TestRun(t *testing.T) {
	var cmd string

	// case 1: uses a simple command.
	cmd = "echo Hello world"
	if err := forest.run(cmd, 30); err != nil {
		t.Errorf("Command '%s' should have worked", cmd)
	}

	// case 2: uses a simple command but with an unsuitable timeout.
	cmd = "sleep 5"
	if err := forest.run(cmd, 0); err == nil {
		t.Errorf("Command '%s' should not have worked", cmd)
	}

	// case 3: uses a broken command.
	cmd = "helloworld"
	if err := forest.run(cmd, 30); err == nil {
		t.Errorf("Command '%s' should not have worked", cmd)
	}
}

func TestUnconv(t *testing.T) {
	var file *gfile.File

	workingDir := "test"
	os.Mkdir(workingDir, 0666)

	loadCommandConfigs("../../../_tests/configurations/gotenberg.yml")

	// case 1: uses an Markdown file type.
	file = makeFile(workingDir, "file.md")
	if _, err := Unconv(workingDir, file); err != nil {
		t.Errorf("Converting '%s' to PDF should have worked", file.Path)
	}

	// case 2: uses an HTML file type.
	file = makeFile(workingDir, "file.html")
	if _, err := Unconv(workingDir, file); err != nil {
		t.Errorf("Converting '%s' to PDF should have worked", file.Path)
	}

	// case 3: uses an Office file type.
	file = makeFile(workingDir, "file.docx")
	if _, err := Unconv(workingDir, file); err != nil {
		t.Errorf("Converting '%s' to PDF should have worked", file.Path)
	}

	// case 4: uses a PDF file type.
	file = makeFile(workingDir, "file.pdf")
	if _, err := Unconv(workingDir, file); err == nil {
		t.Errorf("Converting '%s' to PDF should not have worked", file.Path)
	}

	loadCommandConfigs("../../../_tests/configurations/timeout-gotenberg.yml")

	// case 5: uses a command with an unsuitable timeout.
	file = makeFile(workingDir, "file.docx")
	if _, err := Unconv(workingDir, makeFile(workingDir, "file.docx")); err == nil {
		t.Errorf("Converting '%s' to PDF should have reached timeout", file.Path)
	}

	os.RemoveAll(workingDir)
}

func TestMerge(t *testing.T) {
	workingDir := "test"
	os.Mkdir(workingDir, 0666)

	loadCommandConfigs("../../../_tests/configurations/gotenberg.yml")

	var filesPaths []string
	path, _ := filepath.Abs("../../../_tests/file.pdf")
	filesPaths = append(filesPaths, path)
	filesPaths = append(filesPaths, path)

	// case 1: simple merge.
	if _, err := Merge(workingDir, filesPaths); err != nil {
		t.Error("Merge should have worked")
	}

	loadCommandConfigs("../../../_tests/configurations/timeout-gotenberg.yml")

	// case 2: uses a command with an unsuitable timeout.
	if _, err := Merge(workingDir, filesPaths); err == nil {
		t.Error("Merge should have reached timeout")
	}

	os.RemoveAll(workingDir)
}

func TestImpossibleConversionError(t *testing.T) {
	err := &impossibleConversionError{}
	if err.Error() != impossibleConversionErrorMessage {
		t.Errorf("Error returned a wrong message: got '%s' want '%s'", err.Error(), impossibleConversionErrorMessage)
	}
}

func TestCommandTimeoutError(t *testing.T) {
	err := &commandTimeoutError{
		command: "echo hello",
		timeout: 30,
	}
	expected := fmt.Sprintf("The command '%s' has reached the %d second(s) timeout", err.command, err.timeout)
	if err.Error() != expected {
		t.Errorf("Error returned a wrong message: got '%s' want '%s'", err.Error(), expected)
	}
}
