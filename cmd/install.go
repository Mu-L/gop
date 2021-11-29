/*
 * Copyright (c) 2021 The GoPlus Authors (goplus.org). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func getcwd() string {
	path, _ := os.Getwd()
	return path
}

func getGopLocalLink() string {
	path, _ := os.UserHomeDir()
	return filepath.Join(path, "gop")
}

var gopRoot = getcwd()
var gopLocalLink = getGopLocalLink()

func checkPathExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func execCommand(command string, arg ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(command, arg...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func getGitInfo() (string, string) {
	gitDir := filepath.Join(gopRoot, ".git")
	noBranch := "nobranch"
	noCommit := "nocommit"

	if checkPathExist(gitDir) {
		branch, stderr, _ := execCommand("git", "branch", "--show-current")
		if len(stderr) > 0 {
			println(stderr)
			branch = noBranch
		} else {
			branch = strings.TrimRight(branch, "\n")
		}

		commit, stderr, _ := execCommand("git", "rev-parse", "--verify", "HEAD")
		if len(stderr) > 0 {
			println(stderr)
			commit = noCommit
		} else {
			commit = strings.TrimRight(commit, "\n")
		}
		return branch, commit
	}
	return noBranch, noCommit
}

func getBuildDateTime() string {
	now := time.Now()
	return now.Format("2006-01-02_15-04-05")
}

func getGopBuildFlags() string {
	branch, commit := getGitInfo()
	buildDateTime := getBuildDateTime()

	buildFlags := fmt.Sprintf("-X github.com/goplus/gop/env.buildDate=%s ", buildDateTime)
	buildFlags += fmt.Sprintf("-X github.com/goplus/gop/env.buildCommit=%s ", commit)
	buildFlags += fmt.Sprintf("-X github.com/goplus/gop/env.buildBranch=%s", branch)
	return buildFlags
}

func detectGoBinPath() string {
	goBin, ok := os.LookupEnv("GOBIN")
	if ok {
		return goBin
	}

	goPath, ok := os.LookupEnv("GOPATH")
	if ok {
		return filepath.Join(goPath, "bin")
	}

	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "go", "bin")
}

func buildGoplusTools() {
	commandsDir := filepath.Join(gopRoot, "cmd")
	if !checkPathExist(commandsDir) {
		println("Error: This script should be run at the root directory of gop repository.")
		os.Exit(1)
	}

	buildFlags := getGopBuildFlags()
	goBinPath := detectGoBinPath()

	// If same name file exists, backup it.
	cmdBinPath := filepath.Join(goBinPath, "cmd")
	cmdBackupBinPath := filepath.Join(goBinPath, "cmd-backup-gop")
	if checkPathExist(cmdBinPath) {
		os.Rename(cmdBinPath, cmdBackupBinPath)
	}

	println("Installing Go+ tools...")
	os.Chdir(commandsDir)
	buildOutput, buildErr, err := execCommand("go", "install", "-v", "-ldflags", buildFlags, "./...")
	println(buildErr)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	println(buildOutput)

	// Remove unwanted cmd binary file.
	if checkPathExist(cmdBinPath) {
		os.Remove(cmdBinPath)
	}
	// Restore the backup file if exists.
	if checkPathExist(cmdBackupBinPath) {
		os.Rename(cmdBackupBinPath, cmdBinPath)
	}

	println("Go+ tools installed successfully!")
}

func linkGoplusToLocal() {
	fmt.Printf("Linking %s to %s\n", gopRoot, gopLocalLink)

	os.Chdir(gopRoot)
	if gopLocalLink != gopRoot && !checkPathExist(gopLocalLink) {
		err := os.Symlink(gopRoot, gopLocalLink)
		if err != nil {
			println(err.Error())
		}
	}

	fmt.Printf("%s linked to %s successfully!\n", gopRoot, gopLocalLink)
}

func runTestcases() {
	println("Start running testcases.")
	os.Chdir(gopRoot)

	path, _ := os.LookupEnv("PATH")
	path = fmt.Sprintf("%s:", detectGoBinPath()) + path

	cmd := exec.Command("gop", "test", "-v", "-coverprofile=coverage.txt", "-covermode=atomic", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "PATH="+path)
	err := cmd.Run()
	if err != nil {
		println(err.Error())
	}
	println("End running testcases.")
}

func localInstall() {
	buildGoplusTools()
	linkGoplusToLocal()
	println("Go+ is now installed.")
}

func localUninstall() {
	println("Uninstalling Go+ and related tools.")

	goBinPath := detectGoBinPath()
	filesToRemove := []string{
		gopLocalLink,
		filepath.Join(goBinPath, "gop"),
		filepath.Join(goBinPath, "gopfmt"),
		filepath.Join(goBinPath, "goptestgo"),
	}

	for _, file := range filesToRemove {
		if !checkPathExist(file) {
			continue
		}
		if err := os.Remove(file); err != nil {
			println(err.Error())
		}
	}

	println("Go+ and related tools uninstalled successfully.")
}

func main() {
	isInstall := flag.Bool("install", false, "Install Go+")
	isBuild := flag.Bool("build", false, "Build the Go+")
	isTest := flag.Bool("test", false, "Run testcases")
	isUninstall := flag.Bool("uninstall", false, "Uninstall Go+")

	flag.Parse()

	flagActionMap := map[*bool]func(){
		isInstall:   localInstall,
		isTest:      runTestcases,
		isBuild:     buildGoplusTools,
		isUninstall: localUninstall,
	}

	for flag, action := range flagActionMap {
		if *flag {
			action()
			return
		}
	}

	println("Usage:\n")
	flag.PrintDefaults()
}