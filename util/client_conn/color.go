package client_conn

import "fmt"

const (
	FormatRed    = "\033[31m%s\033[0m"
	FormatGreen  = "\033[32m%s\033[0m"
	FormatYellow = "\033[33m%s\033[0m"
	FormatBlue   = "\033[34m%s\033[0m"
)

func Red(msg string) string {
	return fmt.Sprintf(FormatRed, msg)
}

func Green(msg string) string {
	return fmt.Sprintf(FormatGreen, msg)
}

func Yellow(msg string) string {
	return fmt.Sprintf(FormatYellow, msg)
}

func Blue(msg string) string {
	return fmt.Sprintf(FormatBlue, msg)
}
