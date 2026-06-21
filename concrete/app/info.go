// Package app provides shared application lifecycle logic and bootstrap helpers.
package app

// Info identifies the running application.
type Info struct {
	Name        string
	Version     string
	Environment string
	Debug       bool
}
