package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
)

func getCommandOutput(cmdName string, cmdArgs string) string {
	output, _ := runCommand(cmdName, cmdArgs, false, false)
	return output
}

func getCommandExitCode(cmdName string, cmdArgs string) int {
	_, exit := runCommand(cmdName, cmdArgs, false, true) // Sends quiet signal
	return exit
}

func getCommandOutputAndExitCode(cmdName string, cmdArgs string) (string, int) {
	output, exit := runCommand(cmdName, cmdArgs, false, false)
	return output, exit
}

func streamAndGetCommandOutput(cmdName string, cmdArgs string) string {
	output, _ := runCommand(cmdName, cmdArgs, true, false)
	return output
}
func streamAndGetCommandOutputAndExitCode(cmdName string, cmdArgs string) (string, int) {
	output, exit := runCommand(cmdName, cmdArgs, true, false)
	return output, exit
}

func streamAndGetCommandExitCode(cmdName string, cmdArgs string) int {
	_, exit := runCommand(cmdName, cmdArgs, true, false)
	return exit
}

type output struct {
	buf         *bytes.Buffer
	stream      bool
	combinedOut *combinedOutput
}

func (o *output) Write(p []byte) (int, error) {
	if o.stream {
		splitByNewline := strings.Split(strings.Trim(string(p), "\n"), "\n")
		for _, l := range splitByNewline {
			fmt.Println("\t| ", l)
		}
	}
	o.combinedOut.Write(string(p))
	return o.buf.Write(p)
}

type combinedOutput struct {
	lines []string
}

func (c *combinedOutput) Write(s string) {
	c.lines = append(c.lines, s)
}

func runCommand(cmdName string, cmdArgs string, stream bool, quiet bool) (string, int) {

	// This cmdArgs mess is to facilitate running arbitrary shell commands via `bash -c "<command>"`
	// Regex will split into groups either by whitespace or by quotation marks
	splitRe := regexp.MustCompile(`"(.+)"|(\S+)`)
	brokenArgs := splitRe.FindAllString(cmdArgs, -1)
	// Then, remove the quotation marks
	for i, s := range brokenArgs {
		if strings.Contains(s, "\"") {
			brokenArgs[i] = strings.Replace(s, "\"", "", -1)
		}
	}

	cmd := exec.Command(cmdName, brokenArgs...)

	combinedOutput := &combinedOutput{
		lines: []string{},
	}
	var sout = &output{
		buf:         &bytes.Buffer{},
		stream:      stream,
		combinedOut: combinedOutput,
	}
	cmd.Stdout = sout

	var serr = &output{
		buf:         &bytes.Buffer{},
		stream:      stream,
		combinedOut: combinedOutput,
	}
	cmd.Stderr = serr

	if err := cmd.Start(); err != nil {
		fmt.Fprint(
			os.Stderr,
			"=> There was an error starting command: `",
			cmdName, " ",
			strings.Trim(fmt.Sprint(cmdArgs), "[]"),
			"`, resulting in the error: ",
			err,
			"\n")
		if !stream {
			fmt.Println("\n\t| ", strings.Join(combinedOutput.lines, "\n\t| "))
		}
		fmt.Printf("\n")
		os.Exit(1)
	}

	var exitCode int
	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, exitStatus := exiterr.Sys().(syscall.WaitStatus); exitStatus {
				exitCode = status.ExitStatus()
			}
		}
		if !quiet {
			fmt.Fprint(
				os.Stderr,
				"=> There was an error while running command: `",
				cmdName, " ",
				strings.Trim(fmt.Sprint(cmdArgs), "[]"),
				"`, resulting in the error: ",
				err,
				"\n")
			if !stream {
				fmt.Println("\n\t| ", strings.Join(combinedOutput.lines, "\n\t| "))
			}
			fmt.Printf("\n")
		}
	}

	return strings.Join(combinedOutput.lines, "\n"), exitCode
}
