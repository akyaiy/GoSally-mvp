package logs

import "fmt"

func SetBlack(s string) string   { return fmt.Sprintf("\033[30m%s\033[0m", s) }
func SetRed(s string) string     { return fmt.Sprintf("\033[31m%s\033[0m", s) }
func SetGreen(s string) string   { return fmt.Sprintf("\033[32m%s\033[0m", s) }
func SetYellow(s string) string  { return fmt.Sprintf("\033[33m%s\033[0m", s) }
func SetBlue(s string) string    { return fmt.Sprintf("\033[34m%s\033[0m", s) }
func SetMagenta(s string) string { return fmt.Sprintf("\033[35m%s\033[0m", s) }
func SetCyan(s string) string    { return fmt.Sprintf("\033[36m%s\033[0m", s) }
func SetWhite(s string) string   { return fmt.Sprintf("\033[37m%s\033[0m", s) }

func SetBrightBlack(s string) string   { return fmt.Sprintf("\033[90m%s\033[0m", s) }
func SetBrightRed(s string) string     { return fmt.Sprintf("\033[91m%s\033[0m", s) }
func SetBrightGreen(s string) string   { return fmt.Sprintf("\033[92m%s\033[0m", s) }
func SetBrightYellow(s string) string  { return fmt.Sprintf("\033[93m%s\033[0m", s) }
func SetBrightBlue(s string) string    { return fmt.Sprintf("\033[94m%s\033[0m", s) }
func SetBrightMagenta(s string) string { return fmt.Sprintf("\033[95m%s\033[0m", s) }
func SetBrightCyan(s string) string    { return fmt.Sprintf("\033[96m%s\033[0m", s) }
func SetBrightWhite(s string) string   { return fmt.Sprintf("\033[97m%s\033[0m", s) }
