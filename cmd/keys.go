package cmd

import (
	"fmt"
	"os"
)

func runKeys() error {
	if len(os.Args) < 3 {
		fmt.Println("Usage: dispatch keys <create|list|revoke> [options]")
		return nil
	}

	// TODO: implement key management
	fmt.Println("Key management coming in v0.2")
	return nil
}
