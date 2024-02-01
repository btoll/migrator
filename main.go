package main

import (
	"flag"
)

func main() {
	file := flag.String("file", "repos.txt", "The name of the file that contains the repositories to clone")
	project := flag.String("project", "aion", "The name of the project")
	flag.Parse()

	m := NewMigrator(*file, *project)
	m.migrate(*file, *project)
}
