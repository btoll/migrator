package color

import "fmt"

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Purple = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"

func Branch(s string) string {
	return fmt.Sprintf("`%s%s%s`", Green, s, Reset)
}

func Error() string {
	return fmt.Sprintf("%s[ERROR]%s", Red, Reset)
}

func Info() string {
	return fmt.Sprintf("%s[INFO]%s", Purple, Reset)
}

func Repository(s string) string {
	return fmt.Sprintf("`%s%s%s`", Gray, s, Reset)
}

func Success() string {
	return fmt.Sprintf("%s[SUCCESS]%s", Green, Reset)
}

func Warning() string {
	return fmt.Sprintf("%s[WARNING]%s", Yellow, Reset)
}
