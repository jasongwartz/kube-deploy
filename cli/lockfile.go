package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

const locksRootPath string = "/kube-deploy/locks/"

type lockFileContents struct {
	Author      string
	Reason      string
	DateStarted string
}

func lockFileExists(filename string) bool {

	// Check if locks directory exists
	if _, err := os.Stat(locksRootPath); os.IsNotExist(err) {
		// If not, make the directory and return false (lockfiles can't exist if the directory doesn't)
		os.MkdirAll(locksRootPath, 0777)
		return false
	}

	// Check if lock file exists - error if not
	if _, err := os.Stat(locksRootPath + filename); os.IsNotExist(err) {
		return false
	}
	return true
}

func readLockFile(filename string) lockFileContents {
	fileBytes, err := ioutil.ReadFile(locksRootPath + filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed reading repo config file:", err)
		os.Exit(1)
	}

	lockFileData := lockFileContents{}
	if err := json.Unmarshal(fileBytes, &lockFileData); err != nil {
		panic(err)
	}
	return lockFileData
}

func WriteLockFile(filename, reason string) {
	currentUser := os.Getenv("USER")
	lockFileData := lockFileContents{
		Author:      currentUser,
		Reason:      reason,
		DateStarted: time.Now().Format("Jan _2 15:04:05"),
	}
	jsonBytes, err := json.Marshal(lockFileData)
	if err != nil {
		panic(err.Error())
	}
	err = ioutil.WriteFile(locksRootPath+filename, jsonBytes, 0666)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("=> Successfully wrote lockfile for '%s'.\n\n", filename)
}

func DeleteLockFile(filename string) {
	err := os.Remove(locksRootPath + filename)
	if err != nil {
		panic(err.Error())
	}
}

func IsLocked(applicationName string) bool {
	if lockFileExists("all") {
		fmt.Println("=> All rollouts are currently blocked.")
		lock := readLockFile("all")
		fmt.Printf("\tBlocked by: %s\n\tFor reason: %s\n\tOn date: %s\n",
			lock.Author, lock.Reason, lock.DateStarted)
		return true
	}
	if lockFileExists(applicationName) {
		fmt.Printf("=> Rollouts for %s are blocked.\n", applicationName)
		lock := readLockFile(applicationName)
		fmt.Printf("\tBlocked by: %s\n\tFor reason: %s\n\tOn date: %s\n",
			lock.Author, lock.Reason, lock.DateStarted)
		return true
	}
	return false
}

func LockBeforeRollout(applicationName string, force bool) {
	if !IsLocked(applicationName) {
		WriteLockFile(applicationName, "rollout in progress")
	} else {
		if force {
			fmt.Println("=> Lockfile exists, but proceeding anyway due to '--force'.")
			return
		}
		os.Exit(1)
	}
}

func UnlockAfterRollout(applicationName string) {
	DeleteLockFile(applicationName)
}
