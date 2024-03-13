package main

import (
	"bufio"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/btoll/migrator/color"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"gopkg.in/yaml.v3"
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

type Service struct {
	Name        string
	Environment string
	Image       string // TODO
	Resources   []string
	HasIngress  *string
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
	UseLogin        bool
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
				"error":       ServiceNames{},
				"master":      ServiceNames{},
				"development": ServiceNames{},
				"noKube":      ServiceNames{},
			},
		},
		Dirs: &BuildDirs{
			Build:                    "build",
			Project:                  fmt.Sprintf("%s/%s", "build", project.Name),
			Cloned:                   "build/cloned",
			AnsibleDeployers:         "build/ansible-deployers",
			AnsibleDeployerOverrides: "build/ansible-deployers/files/kubernetes_environment_overrides",
		},
	}
}

func (m *Migrator) clone(serviceName string) {
	clonedAppDir := fmt.Sprintf("%s/%s/%s", m.Dirs.Cloned, m.Project.Name, serviceName)
	tmpClonedDir := fmt.Sprintf("%s", clonedAppDir)

	// Get the `development` branch first, if there is one and fall back to the `master` branch.
	// NOTE: The `go-git` library wasn't returning all of the branches.
	var branchName plumbing.ReferenceName
	branchName = "refs/heads/development"
	_, err := git.PlainClone(tmpClonedDir, false, &git.CloneOptions{
		URL:           fmt.Sprintf("git@bitbucket.org:pecteam/%s", serviceName),
		Progress:      nil,
		ReferenceName: branchName,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("%s Could not clone the `%s` branch for the `%s` repository, trying master...", color.Warning(), branchName, serviceName))
		branchName = "refs/heads/master"
		_, err = git.PlainClone(tmpClonedDir, false, &git.CloneOptions{
			URL:           fmt.Sprintf("git@bitbucket.org:pecteam/%s", serviceName),
			Progress:      nil,
			ReferenceName: branchName,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("%s Could not clone the `%s` branch for the `%s` repository", color.Warning(), branchName, serviceName))
			fmt.Printf("%s err %s\n", serviceName, err)
			m.Debug.Files["error"] = append(m.Debug.Files["error"], serviceName)
		} else {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("   %s Cloned the %s branch for the %s repository", color.Info(), color.Branch(string(branchName)), color.Repository(serviceName)))
			m.Debug.Files["master"] = append(m.Debug.Files["master"], serviceName)
		}
	} else {
		m.Debug.Files["development"] = append(m.Debug.Files["development"], serviceName)
		fmt.Fprintln(os.Stderr, fmt.Sprintf("   %s Cloned the %s branch for the %s repository", color.Info(), color.Branch(string(branchName)), color.Repository(serviceName)))
	}

	ansibleDeployersDir := fmt.Sprintf("%s/ansible-deployers", m.Dirs.Build)
	if !checkFileExists(ansibleDeployersDir) {
		_, err := git.PlainClone(ansibleDeployersDir, false, &git.CloneOptions{
			URL:      "git@bitbucket.org:pecteam/ansible-deployers.git",
			Progress: nil,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("%s Could not clone `ansible-deployers`", color.Error()))
			log.Fatal(err)
		}
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

func (m *Migrator) kustomize() {
	// Get a list of all services that have been cloned to `build/{PROJECT_NAME}`.
	// These need to be tricked out for Kustomize.  The directory structure we'll
	// be using is:
	//
	//	aion-nginx/
	//	├── base/
	//	│   ├── RESOURCES (manifests)
	//	│   ├── env
	//	│   └── kustomization.yaml
	//	└── overlays/
	//		├── beta/
	//		│   ├── env
	//		│   └── kustomization.yaml
	//		├── development/
	//		│   ├── env
	//		│   └── kustomization.yaml
	//		└── production/
	//			├── env
	//			└── kustomization.yaml
	//
	dirs, err := os.ReadDir(fmt.Sprintf("%s/%s", m.Dirs.Cloned, m.Project.Name))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not list contents of the build directory")
		//		log.Fatal(err)
	}
	foregroundServicesFile := fmt.Sprintf("%s/vars/aion_foreground_services.yml", m.Dirs.AnsibleDeployers)
	foregroundServices, err := os.ReadFile(foregroundServicesFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("%s Could not read file `%s`", color.Warning(), foregroundServicesFile))
		//				log.Fatal(err)
	}

	for _, dir := range dirs {
		repo := dir.Name()
		clonedAppDir := fmt.Sprintf("%s/%s/%s", m.Dirs.Cloned, m.Project.Name, repo)
		appDir := fmt.Sprintf("%s/%s", m.Dirs.Project, repo)
		kubeDir := fmt.Sprintf("%s/.kube", clonedAppDir)

		if !checkFileExists(kubeDir) {
			m.Debug.Files["noKube"] = append(m.Debug.Files["noKube"], repo)
			continue
		}

		// Create the build/serviceNameDir and Kustomized scaffolding for the service (repo).
		m.scaffold(appDir)

		// Tokenize and create the Kubernetes manifests, writing them to `base/`.
		k := Service{Name: repo}

		// Get all dir entries in ".kube" and replace all tokens (`{{ TOKEN_NAME }}`) that we
		// can in the Kubernetes manifest Jinja template files.  Some tokens will not be able
		// to be replaced, as they are in the `ansible-deployers` repo.  This will be addressed
		// in a later step.
		//
		// These will only replace the values gotten from the default environments file.  The
		// other environment-specific values in `ansible-deployers` will be used to tokenize the
		// special-case manifests such as Ingress and write them to the Kustomize overlays directory.
		files, err := os.ReadDir(kubeDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("%s Could not list contents of the .kube directory", color.Error()))
			//			log.Fatal(err)
		}
		for _, f := range files {
			filename := f.Name()
			if filepath.Ext(filename) == m.TplExt {
				var content []byte
				f := fmt.Sprintf("%s/%s", kubeDir, filename)
				// Maybe fix up the manifests after variable substitution?
				if strings.Contains(filename, "deployment") {
					content = hackAndBeHappyDeployment(f, repo)
					//				} else if strings.Contains(filename, "ingress") {
					//					content = hackAndBeHappyIngress(f)
				} else {
					content, err = os.ReadFile(f)
					if err != nil {
						fmt.Fprintln(os.Stderr, fmt.Sprintf("%s Could not read file `%s`", color.Warning(), f))
						//						log.Fatal(err)
					}
				}

				// Remove the extension ONLY after the template file has been read.
				filename = strings.TrimSuffix(filename, filepath.Ext(filename))

				// A serviceName name will usually (always?) contain a hyphen (`-`), and this
				// must be removed to match the filenames in the `.kube` directory.
				// For example:
				// The `aion-nginx` serviceName contains (among others):
				//		aionnginx-deployment.yaml.j2
				//		defaults-aionnginx.yaml
				//		beta-aionnginx.yaml
				// EXCEPTIONS...
				// The `aion-alert-micro` (and others) doesn't follow this nice little
				// pattern.  Instead, its Jinja template files look like this:
				// 		aionalertconnectionsconsumer-deployment.yaml.j2
				// 		aionalertintegrationsconsumer-deployment.yaml.j2
				// 		aionalertmicro-deployment.yaml.j2
				// 		aionalertmicro-service.yaml.j2
				// In other words, there are files that are named after services
				// other than `aion-alert-micro`.
				// SO...
				// We cannot depend on the service name.  Instead, get the name up to
				// the hyphen and match it against each `default-` environment file
				// in `environments`.

				// TODO
				// secrets_reader_config_map: kubernetes-container-user

				before, _, _ := strings.Cut(filename, "-")
				defaultManifestValues := fmt.Sprintf("%s/environments/defaults-%s.yaml", kubeDir, before)
				tokenized := tokenizeManifests(getManifestValues(defaultManifestValues), string(content))

				// This is just fucking horrible.  Since `ansible-deployers` was injecting this into the
				// manifest via a Python script, there's not a "nice" way to do it here except by doing
				// the same horrible thing (injection).
				// If it were part of the templated Jinja manifest it wouldn't be nearly as ugly as this.
				if strings.Contains(filename, "deployment") {
					// We need to append the `nodeSelector` label to the tokenized string.
					var nodeSelectorLabel string
					if strings.Contains(string(foregroundServices), repo) {
						nodeSelectorLabel = "node_type: application"
					} else {
						nodeSelectorLabel = "node_type: default"
					}
					tokenized = fmt.Sprintf("%s      nodeSelector:\n        %s", tokenized, nodeSelectorLabel)
				}

				// We're going to patch the Ingress, so don't include it in the list
				// that will become the `resources` list in base/kustomization.yaml.
				// Instead, flag it so we know to add it as `overlays/ENV/patch_ingress.yaml`
				// (see below).
				if !strings.Contains(filename, "ingress") {
					k.Resources = append(k.Resources, filename)
					writeManifestFile(filename, tokenized, appDir)
				} else {
					k.HasIngress = &tokenized
				}
			}
		}

		// Create the base `kustomization.yaml`.
		f, err := os.Create(fmt.Sprintf("%s/base/kustomization.yaml", appDir))
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Could not create %s/base/kustomization.yaml", appDir))
			//			log.Fatal(err)
		}
		defer f.Close()

		err = m.Template.ExecuteTemplate(f, "kustomization_base.tpl", k)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Could not execute template `kustomization_base.tpl`")
			//			log.Fatal(err)
		}

		// Create the `env` file in each overlays environment that will be used to generate the
		// ConfigMap that will replace the embedding of the env vars in the Deployment.
		// In addition, create the `kustomization.yaml` file in each overlays environment dir.
		//
		// Note: if the service includes an Ingress, then include this as a patch in the respective
		// overlay environment's directory as a patch.
		for _, env := range m.Environments {
			// Create a `kustomization.yaml` for each environment in overlays.
			f, err = os.Create(fmt.Sprintf("%s/overlays/%s/kustomization.yaml", appDir, env))
			if err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("Could not create %s/overlays/%s/kustomization.yaml", appDir))
				//				log.Fatal(err)
			}
			defer f.Close()

			k.Environment = env
			// TODO need the k.Image name!
			k.Image = "TODO"
			err = m.Template.ExecuteTemplate(f, "kustomization_overlay.tpl", k)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Could not execute template `kustomization_overlay.tpl`")
				//				log.Fatal(err)
			}

			// The environment file in .kube won't necessarily match the name of the service or
			// repository.  However, we may be able to get away with just grabbing the env vars
			// from the name of the environment that DOES match the service name, since the env
			// vars may be close to the same for every environment. TODO
			var serviceName string
			if strings.Contains(repo, "-") {
				serviceName = strings.ReplaceAll(repo, "-", "")
			}
			envFile := fmt.Sprintf("%s/environments/%s-%s.yaml", kubeDir, env, serviceName)
			content, err := os.ReadFile(envFile)
			if err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("%s Could not read file `%s`", color.Warning(), envFile))
				//				log.Fatal(err)
			}
			baseEnvVars := T{}
			err = yaml.Unmarshal(content, &baseEnvVars)
			if err != nil {
				//				log.Fatal(err)
			}

			// overrides in ansible-deployers
			// 		  repo = aion-nginx
			// serviceName = aionnginx
			overridesFile := fmt.Sprintf("%s/%s/environments/%s-%s.yaml", m.Dirs.AnsibleDeployerOverrides, repo, env, serviceName)
			content, _ = os.ReadFile(overridesFile)
			if err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("%s Could not read file `%s`", color.Warning(), overridesFile))
				//				log.Fatal(err)
			}
			overridesEnvVars := T{}
			err = yaml.Unmarshal(content, &overridesEnvVars)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Could not unmarshal `ovveridesEnvVars`")
				//				log.Fatal(err)
			}

			envvars := map[string]string{}
			replaceMerge(envvars, baseEnvVars, overridesEnvVars)

			f, err := os.Create(fmt.Sprintf("%s/overlays/%s/env", appDir, env))
			if err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("Could not create file %s/overlays/%s/env", appDir, env))
				//				log.Fatal(err)
			}
			defer f.Close()
			err = m.Template.ExecuteTemplate(f, "env.tpl", envvars)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Could not execute template `env.tpl`")
				//				log.Fatal(err)
			}

			if k.HasIngress != nil {
				baseFile := fmt.Sprintf("%s/environments/%s-%s.yaml", kubeDir, k.Environment, serviceName)
				overridesFile = fmt.Sprintf("%s/%s/environments/%s-%s.yaml", m.Dirs.AnsibleDeployerOverrides, repo, k.Environment, serviceName)

				mergedValues := mapMerge(
					getManifestValues(baseFile),
					getManifestValues(overridesFile),
				)
				tokenized := tokenizeManifests(mergedValues, *k.HasIngress)

				// TODO: aion-multi-system-login-nginx has `additional_certs` in its `mergedValues` map BUT
				// is looking for `additional_certificates` in its tokenized Ingress manifest.
				//				if k.Environment == "beta" {
				//					fmt.Println("mergedValues", mergedValues)
				//					fmt.Println("tokenized", tokenized)
				//				}

				// This whole block is just horrid.
				//
				// Check the newer dictionary first.
				// See `ansible-deployers/vars/certificates.yml`.
				// Get some certificate values, without which nothing can be looked up.
				apexDomain, applicationRegion, err := getCertificateValues(mergedValues)
				if err != nil {
					fmt.Println(fmt.Sprintf("  %s %s", color.Error(), err))
				}
				contents, err := os.ReadFile("build/ansible-deployers/vars/certificates.yml")
				if err != nil {
					fmt.Println(fmt.Sprintf("%s Could not read the certificates file", color.Error()))
				}
				c := &C{}
				err = yaml.Unmarshal(contents, &c)
				if err != nil {
					fmt.Println(err)
				}
				certsByRegion, certsOK := c.CertificatesByDomainAndRegion[apexDomain]
				if certsOK {
					cert, certOK := certsByRegion[applicationRegion]
					if certOK {
						tokenized = strings.ReplaceAll(tokenized, "{{ certificates_by_domain_and_region[apex_domain][application_region] }}", cert)
					} else {
						fmt.Println(fmt.Sprintf("%s There is no application region certification", color.Warning()))
					}
				} else {
					fmt.Println(fmt.Sprintf("%s There are no certifications by application region", color.Warning()))
					// Fall back to the old method.
					// See `ansible-deployers/vars/certificates.yml`.
					cert, certOK := c.Certificates[apexDomain]
					if certOK {
						tokenized = strings.ReplaceAll(tokenized, "{{ certificates[apex_domain] }}", cert)
					} else {
						fmt.Println(fmt.Sprintf("%s There is no certification", color.Warning()))
					}
				}

				_, additionalCertsOK := mergedValues["additional_certs"]
				_, additionalCertificatesOK := mergedValues["additional_certificates"]
				var hasAdditionalCerts bool
				if additionalCertsOK || additionalCertificatesOK {
					hasAdditionalCerts = true
				}

				// Now we want to fix up the Ingress manifest template.  We do it now because we have access
				// to the variables that we need to make our determinations about what to do with the template.
				lines := hackAndBeHappyIngress(tokenized, k.Environment, hasAdditionalCerts)

				ingressFile := fmt.Sprintf("%s/overlays/%s/ingress.yaml", appDir, k.Environment)
				fd, err := os.Create(ingressFile)
				w := bufio.NewWriter(fd)
				for _, line := range lines {
					w.WriteString(fmt.Sprintf("%s\n", string(line)))
				}
				w.Flush()
			}
		}

		err = os.RemoveAll(kubeDir)
		if err != nil {
			fmt.Println(err, fmt.Sprintf("%s Could not remove the %s", color.Warning(), kubeDir))
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

	m.kustomize()
	m.debug()
}

func (m *Migrator) scaffold(appDir string) {
	err := os.Mkdir(appDir, os.ModePerm)
	if err != nil {
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
