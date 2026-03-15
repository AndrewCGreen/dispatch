package cmd

import "fmt"

func runDoctor() error {
	fmt.Println("dispatch doctor — checking your setup...\n")

	// TODO: implement checks
	// - Config file exists and parses
	// - Database is accessible
	// - Each site config is valid
	// - Backend connectivity
	// - DNS records (SPF, DKIM, DMARC) for each site's sending domain

	fmt.Println("Doctor checks coming in v0.2")
	return nil
}
