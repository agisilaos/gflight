package cli

import "fmt"

func (a App) help(_ []string) error {
	fmt.Print(usageText())
	return nil
}
