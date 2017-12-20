package main

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

func writeLockFile(filename, reason string) {
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

func deleteLockFile(filename string) {
	err := os.Remove(locksRootPath + filename)
	if err != nil {
		panic(err.Error())
	}
}

func isLocked() bool {
	if lockFileExists("all") {
		fmt.Println("=> All rollouts are currently blocked.")
		lock := readLockFile("all")
		fmt.Printf("\tBlocked by: %s\n\tFor reason: %s\n\tOn date: %s\n",
			lock.Author, lock.Reason, lock.DateStarted)
		return true
	}
	if lockFileExists(repoConfig.Application.Name) {
		fmt.Printf("=> Rollouts for %s are blocked.\n", repoConfig.Application.Name)
		lock := readLockFile(repoConfig.Application.Name)
		fmt.Printf("\tBlocked by: %s\n\tFor reason: %s\n\tOn date: %s\n",
			lock.Author, lock.Reason, lock.DateStarted)
		return true
	}
	return false
}

func lockBeforeRollout() {
	if !isLocked() {
		writeLockFile(repoConfig.Application.Name, "rollout in progress")
	} else {
		os.Exit(1)
	}
}

func unlockAfterRollout() {
	deleteLockFile(repoConfig.Application.Name)
}
