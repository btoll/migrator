package main

import (
	"bufio"
	"fmt"
	"html/template"
	"log"
	"os"
)

type RepositoryNames []string

// Find intersection:
// join <(sort errors/1) <(sort errors/2)
// comm -12 <(sort errors/1) <(sort errors/2)

// TODO
// - clean up all the fmt.Sprintf file interpolations.

type ManifestValues map[string]interface{}

type C struct {
	Certificates                  Cert            `yaml:"certificates"`
	CertificatesByDomainAndRegion map[string]Cert `yaml:"certificates_by_domain_and_region"`
}

type Cert map[string]string

type T struct {
	EnvironmentVariables []EnvironmentVariable `yaml:"environment_variables"`
}

type EnvironmentVariable struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// This is **not** a Kubernetes service!
type Service struct {
	Name          string
	NameNoHyphens string
	Environment   string
	Image         *Image
	Replicas      int
	Resources     []string
	HasIngress    *string
}

// Each image will be defined in `overlays/ENVIRONMENT/kustomization.yaml`.
type Image struct {
	Name    string
	NewName string
	NewTag  string
}

type BuildDirs struct {
	Build                    string
	Project                  string
	Cloned                   string
	AnsibleDeployers         string
	AnsibleDeployerOverrides string
}

type Debug struct {
	Files map[string]ServiceNames
}

type ServiceNames []string

type Migrator struct {
	Project      *Project
	Environments []string
	ReposFile    string
	TplExt       string
	Template     *template.Template
	Dirs         *BuildDirs
	Debug        *Debug
}

type Login struct {
	Username string
	Password string
}

type Project struct {
	Name            string
	Filename        string
	BuildDir        string
	UseLogin        bool
	CloneOnly       bool
	Login           *Login
	RepositoryNames *RepositoryNames
}

func NewMigrator(project *Project) *Migrator {
	tpl, err := template.ParseGlob("tpl/*")
	if err != nil {
		fmt.Println("err", err)
		fmt.Fprintln(os.Stderr, "Could not parse template globs")
		log.Fatalln(err)
	}
	return &Migrator{
		Project:      project,
		Environments: []string{"production", "beta", "development"},
		TplExt:       ".j2",
		Template:     tpl,
		Debug: &Debug{
			Files: map[string]ServiceNames{
				"error":          ServiceNames{},
				"master":         ServiceNames{},
				"development":    ServiceNames{},
				"noKube":         ServiceNames{},
				"noMatchedToken": ServiceNames{},
			},
		},
		Dirs: &BuildDirs{
			Build:                    project.BuildDir,
			Project:                  fmt.Sprintf("%s/%s", project.BuildDir, project.Name),
			Cloned:                   fmt.Sprintf("%s/cloned", project.BuildDir),
			AnsibleDeployers:         fmt.Sprintf("%s/ansible-deployers", project.BuildDir),
			AnsibleDeployerOverrides: fmt.Sprintf("%s/ansible-deployers/files/kubernetes_environment_overrides", project.BuildDir),
		},
	}
}

// This creates a file in `build/` for each of the keys in `m.Debug.Files`
// and is useful to know which service fell into which category.
func (m *Migrator) debug() {
	for filename, v := range m.Debug.Files {
		f, err := os.Create(fmt.Sprintf("%s/%s", m.Dirs.Build, filename))
		if err != nil {
			fmt.Println(err)
		}
		defer f.Close()
		buf := bufio.NewWriter(f)
		for _, name := range v {
			_, err := buf.WriteString(fmt.Sprintf("%s\n", name))
			if err != nil {
				fmt.Println(err)
			}
		}
		if err := buf.Flush(); err != nil {
			fmt.Println(err)
		}
	}
}

func (m *Migrator) migrate() {
	// Create "build/aion".
	err := os.MkdirAll(m.Dirs.Project, os.ModePerm)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not create the build dirs (build/PROJECT_NAME)")
		log.Fatal(err)
	}
	// Create "build/cloned".
	err = os.MkdirAll(m.Dirs.Cloned, os.ModePerm)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not create the cloned dir (cloned/)")
		log.Fatal(err)
	}

	if !m.Project.UseLogin {
		// Contents will never be large enough to need to chunk or buffer.
		readfile, err := os.Open(m.Project.Filename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Could not open repositories file")
			//		log.Fatal(err)
		}
		defer readfile.Close()

		filescanner := bufio.NewScanner(readfile)
		for filescanner.Scan() {
			m.clone(filescanner.Text())
		}
	} else {
		for _, repositoryName := range *m.Project.RepositoryNames {
			m.clone(repositoryName)
		}
	}

	if !m.Project.CloneOnly {
		m.kustomize()
		m.debug()
	}
}

func (m *Migrator) scaffold(appDir string) {
	err := os.Mkdir(appDir, os.ModePerm)
	if err != nil {
		fmt.Println(err)
		fmt.Fprintln(os.Stderr, "Could not create build service directory")
		//		log.Fatal(err)
	}
	// Now, create directory structure for Kustomize.
	err = os.MkdirAll(fmt.Sprintf("%s/base", appDir), os.ModePerm)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not create build service directory")
		//		log.Fatal(err)
	}
	for _, env := range m.Environments {
		err = os.MkdirAll(fmt.Sprintf("%s/overlays/%s", appDir, env), os.ModePerm)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Could not create environment directory in Kustomize overlays directory")
			//			log.Fatal(err)
		}
	}
}
