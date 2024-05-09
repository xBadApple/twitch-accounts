package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/ox-y/GoGmailnator"
)

func fastEmailTest() {
	var sess GoGmailnator.Session

	err := sess.Init(nil)
	if err != nil {
		panic(err)
	}

	isAlive, err := sess.IsAlive()
	if err != nil {
		panic(err)
	}

	if isAlive {
		fmt.Println("Session is alive.")
	} else {
		fmt.Println("Session is dead.")
		return
	}

	emailAddress, err := sess.GenerateEmailAddress()
	if err != nil {
		panic(err)
	}

	fmt.Println("Email address is " + emailAddress + ".")

	emails, err := sess.RetrieveMail(emailAddress)
	if err != nil {
		panic(err)
	}

	for _, email := range emails {
		fmt.Printf("From: %s, Subject: %s, Time: %s\n", email.From, email.Subject, email.Time)
	}
}

func clearScreen() {
	// Clear the screen using platform-specific commands

	switch runtime.GOOS {
	case "linux", "darwin":
		// For Linux and macOS
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	case "windows":
		// For Windows
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		// Unsupported platform
		fmt.Println("Unsupported platform")
	}
}
