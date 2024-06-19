package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/btoll/migrator/color"
	"github.com/ktrysmt/go-bitbucket"
)

func main() {
	filename := flag.String("file", "", "The name of the file that contains the repositories to clone")
	project := flag.String("project", "", "The name of the project")
	buildDir := flag.String("build-dir", "build", "The location of the build directory.  Defaults to `./build`.")
	cloneOnly := flag.Bool("clone-only", false, "Clone but don't kustomize")
	flag.Parse()

	if *project == "" {
		os.Exit(1)
	}

	var p *Project
	if *filename == "" {
		username := os.Getenv("BITBUCKET_USERNAME")
		password := os.Getenv("BITBUCKET_PASSWORD")

		if !(username != "" && password != "") {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("%s Both username (BITBUCKET_USERNAME) and password (BITBUCKET_PASSWORD) must be set.", color.Error()))
			os.Exit(1)
		}

		f, err := os.Create(fmt.Sprintf("repos/%s.names", *project))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer f.Close()
		c := bitbucket.NewBasicAuth(username, password)
		opt := &bitbucket.RepositoriesOptions{
			Owner: "pecteam",
		}
		res, err := c.Repositories.ListForAccount(opt)
		if err != nil {
			panic(err)
		}
		repositoryNames := &RepositoryNames{}
		for _, item := range res.Items {
			if item.Project.Name == *project {
				*repositoryNames = append(*repositoryNames, item.Slug)
				_, err := fmt.Fprintln(f, item.Slug)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
		p = &Project{
			Name:      strings.ToLower(*project),
			BuildDir:  *buildDir,
			UseLogin:  true,
			CloneOnly: *cloneOnly,
			Login: &Login{
				Username: username,
				Password: password,
			},
			RepositoryNames: repositoryNames,
		}
	} else {
		p = &Project{
			Name:      strings.ToLower(*project),
			BuildDir:  *buildDir,
			Filename:  *filename,
			CloneOnly: *cloneOnly,
		}
	}

	startTime := time.Now()
	m := NewMigrator(p)
	m.migrate()
	endTime := time.Now()
	diff := endTime.Sub(startTime)
	fmt.Println("total time taken ", diff.Seconds(), "seconds")
}
