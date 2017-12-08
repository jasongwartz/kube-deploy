package main

import (
	"regexp"
	"fmt"
	//"reflect"
	"strings"
	"os"
	"os/exec"
	"bufio"
	"syscall"
)

func getCommandOutput(cmdName string, cmdArgs string) (string) {
	output, _ := runCommand(cmdName, cmdArgs, false, false)
	return output
}

func getCommandExitCode(cmdName string, cmdArgs string) (int) {
	_, exit := runCommand(cmdName, cmdArgs, false, true) // Sends quiet signal
	return exit
}

func getCommandOutputAndExitCode(cmdName string, cmdArgs string) (string, int) {
	output, exit := runCommand(cmdName, cmdArgs, false, false)
	return output, exit
}

func streamAndGetCommandOutput(cmdName string, cmdArgs string) (string) {
	output, _ := runCommand(cmdName, cmdArgs, true, false)
	return output
}
func streamAndGetCommandOutputAndExitCode(cmdName string, cmdArgs string) (string, int) {
	output, exit := runCommand(cmdName, cmdArgs, true, false)
	return output, exit
}

func streamAndGetCommandExitCode(cmdName string, cmdArgs string) (int) {
	_, exit := runCommand(cmdName, cmdArgs, true, false)
	return exit
}


func runCommand(cmdName string, cmdArgs string, stream bool, quiet bool) (string, int) {

	var (
		cmdOut []string
		cmdError []string
		err error
		exitCode int
	)

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

	cmdReader, _ := cmd.StdoutPipe()
	cmdReadError, _ := cmd.StderrPipe()
	scannerOut := bufio.NewScanner(cmdReader)
	scannerError := bufio.NewScanner(cmdReadError)

	go func() {
		for scannerOut.Scan() {
			if stream { fmt.Printf("\t| %s\n", scannerOut.Text()) }
			cmdOut = append(cmdOut, scannerOut.Text())
		}
		for scannerError.Scan() {
			if stream { fmt.Printf("\t| %s\n", scannerError.Text()) }
			cmdError = append(cmdError, scannerError.Text())
		}
	}()

	if err = cmd.Start(); err != nil {
		fmt.Fprint(
			os.Stderr,
			"=> There was an error starting command: `",
			cmdName, " ",
			strings.Trim(fmt.Sprint(cmdArgs), "[]"), 
			"`, resulting in the error: ",
			err,
			"\n")
		fmt.Println("\n\t| ", strings.Join(cmdOut, "\n"))
		fmt.Println("\n\t| ", strings.Join(cmdError, "\n"))
		os.Exit(1)
	}

	if err = cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, exitStatus := exiterr.Sys().(syscall.WaitStatus); exitStatus {
				exitCode = status.ExitStatus()
			}
		}
		if ! quiet {
			fmt.Fprint(
				os.Stderr,
				"=> There was an error while running command: `",
				cmdName, " ",
				strings.Trim(fmt.Sprint(cmdArgs), "[]"), 
				"`, resulting in the error: ",
				err,
				"\n")
	
			if len(cmdOut) > 0 {
				fmt.Println("\n\t| ", strings.Join(cmdOut, "\n\t| "))
			}
			if len(cmdError) > 0 {
				fmt.Println("\n\t| ", strings.Join(cmdError, "\n\t| "))			
			}
			fmt.Printf("\n")	
		}
	}

	return strings.Join(cmdOut, "\n"), exitCode
}	
