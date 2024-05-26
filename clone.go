package main

import (
	"fmt"
	"log"
	"os"

	"github.com/btoll/migrator/color"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type Cloner struct {
	URL        string
	Repository string
	Branch     string
	CloneDir   string
}

func clone(c *Cloner) (*git.Repository, error) {
	if c.URL == "" {
		c.URL = "git@github.com"
	}
	if c.Branch == "" {
		c.Branch = "master"
	}
	if c.CloneDir == "" {
		c.CloneDir = "."
	}
	if c.Repository == "" {
		fmt.Fprintln(os.Stderr, "Repository cannot be undefined.")
		os.Exit(1)
	}
	return git.PlainClone(c.CloneDir, false, &git.CloneOptions{
		URL:           fmt.Sprintf("%s/%s.git", c.URL, c.Repository),
		Progress:      nil,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", c.Branch)),
	})
}

func (m *Migrator) clone(serviceName string) {
	clonedAppDir := fmt.Sprintf("%s/%s/%s", m.Dirs.Cloned, m.Project.Name, serviceName)
	tmpClonedDir := fmt.Sprintf("%s", clonedAppDir)

	cloner := &Cloner{
		URL:        "git@bitbucket.org:pecteam",
		Repository: serviceName,
		Branch:     "development",
		CloneDir:   tmpClonedDir,
	}

	// Get the `development` branch first, if there is one and fall back to the `master` branch.
	_, err := clone(cloner)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("%s Could not clone the `%s` branch for the `%s` repository, trying master...", color.Warning(), cloner.Branch, cloner.Repository))
		cloner.Branch = "master"
		_, err := clone(cloner)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("%s Could not clone the `%s` branch for the `%s` repository", color.Warning(), cloner.Branch, cloner.Repository))
			fmt.Printf("%s err %s\n", serviceName, err)
			m.Debug.Files["error"] = append(m.Debug.Files["error"], cloner.Repository)
		} else {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("   %s Cloned the %s branch for the %s repository", color.Info(), color.Branch(cloner.Branch), color.Repository(cloner.Repository)))
			m.Debug.Files["master"] = append(m.Debug.Files["master"], cloner.Repository)
		}
	} else {
		m.Debug.Files["development"] = append(m.Debug.Files["development"], serviceName)
		fmt.Fprintln(os.Stderr, fmt.Sprintf("   %s Cloned the %s branch for the %s repository", color.Info(), color.Branch(cloner.Branch), color.Repository(serviceName)))
	}

	ansibleDeployersDir := fmt.Sprintf("%s/ansible-deployers", m.Dirs.Build)
	if !checkFileExists(ansibleDeployersDir) {
		cloner.Repository = "ansible-deployers"
		cloner.Branch = "master"
		cloner.CloneDir = ansibleDeployersDir
		_, err := clone(cloner)
		if err != nil {
			fmt.Println("err", err)
			fmt.Fprintln(os.Stderr, fmt.Sprintf("%s Could not clone `ansible-deployers`", color.Error()))
			log.Fatal(err)
		}
	}
}
