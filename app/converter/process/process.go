// Package process handles all commands executions.
package process

import (
	"bytes"
	"fmt"
	"os/exec"
	"sync"
	"text/template"
	"time"

	"github.com/thecodingmachine/gotenberg/app/config"
	gfile "github.com/thecodingmachine/gotenberg/app/converter/file"
)

type runner struct {
	mu             sync.Mutex
	commandsConfig *config.CommandsConfig
}

var forest = &runner{}

type commandTimeoutError struct {
	command string
	timeout int
}

func (e *commandTimeoutError) Error() string {
	return fmt.Sprintf("The command '%s' has reached the %d second(s) timeout", e.command, e.timeout)
}

// run runs the given command. If timeout is reached or
// something bad happened, returns an error.
func (r *runner) run(command string, timeout int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd := exec.Command("/bin/sh", "-c", command)
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// wait for the process to finish or kill it after a timeout.
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			return err
		}
		return &commandTimeoutError{
			command: command,
			timeout: timeout,
		}
	case err := <-done:
		if err != nil {
			return err
		}

		return nil
	}
}

// Load loads the commands configuration coming from the application configuration.
func Load(config *config.CommandsConfig) {
	forest.commandsConfig = config
}

// conversionData will be applied to the data-driven templates of conversions commands.
type conversionData struct {
	FilePath       string
	ResultFilePath string
}

type impossibleConversionError struct{}

const impossibleConversionErrorMessage = "Impossible conversion"

func (e *impossibleConversionError) Error() string {
	return impossibleConversionErrorMessage
}

// Unconv converts a file to PDF and returns the new file path.
func Unconv(workingDir string, file *gfile.File) (string, error) {
	cmdData := &conversionData{
		FilePath:       file.Path,
		ResultFilePath: gfile.MakeFilePath(workingDir, ".pdf"),
	}

	var (
		cmdTimeout  int
		cmdTemplate *template.Template
	)

	switch file.Type {
	case gfile.MarkdownType:
		cmdTimeout = forest.commandsConfig.Markdown.Timeout
		cmdTemplate = forest.commandsConfig.Markdown.Template
		break
	case gfile.HTMLType:
		cmdTimeout = forest.commandsConfig.HTML.Timeout
		cmdTemplate = forest.commandsConfig.HTML.Template
		break
	case gfile.OfficeType:
		cmdTimeout = forest.commandsConfig.Office.Timeout
		cmdTemplate = forest.commandsConfig.Office.Template
		break
	default:
		return "", &impossibleConversionError{}
	}

	var data bytes.Buffer
	if err := cmdTemplate.Execute(&data, cmdData); err != nil {
		return "", err
	}

	err := forest.run(data.String(), cmdTimeout)
	if err != nil {
		return "", err
	}

	return cmdData.ResultFilePath, nil
}

// mergeData will be applied to the data-driven template of the merge command.
type mergeData struct {
	FilesPaths     []string
	ResultFilePath string
}

// Merge merges many PDF files to one unique PDF file and returns the new file path.
func Merge(workingDir string, filesPaths []string) (string, error) {
	cmdData := &mergeData{
		FilesPaths:     filesPaths,
		ResultFilePath: gfile.MakeFilePath(workingDir, ".pdf"),
	}

	cmdTimeout := forest.commandsConfig.Merge.Timeout
	cmdTemplate := forest.commandsConfig.Merge.Template

	var data bytes.Buffer
	if err := cmdTemplate.Execute(&data, cmdData); err != nil {
		return "", err
	}

	err := forest.run(data.String(), cmdTimeout)
	if err != nil {
		return "", err
	}

	return cmdData.ResultFilePath, nil
}
