package cmd

import (
	"fmt"
	"os"
)

func runSend() error {
	if len(os.Args) < 5 {
		fmt.Println("Usage: dispatch send <site> <template> <email>")
		return nil
	}

	fmt.Printf("Sending %s/%s to %s...\n", os.Args[2], os.Args[3], os.Args[4])
	fmt.Println("Test send coming in v0.2")
	return nil
}
