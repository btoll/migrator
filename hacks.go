package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
)

var reImagePullPolicy = regexp.MustCompile(`^\s*(?P<ImagePullPolicy>imagePullPolicy:\s.*)\s*$`)

// i'm so sorry
func hackAndBeHappyDeployment(filename, repoName string) []byte {
	// Note that sometimes the misspelling `vairable` is spelled correctly, so
	// we'll only look for the portion of the strings that aren't misspelled.
	// For example, here is an example of one of the strings we need to remove:
	// "{% for environment_vairable in environment_vairables %}",
	toRemove := []string{
		"env:",
		"in environment_",
		"- name: \"{{ environment_",
		"value: \"{{ environment_",
		"{% endfor %}",
	}
	toReplace := []string{
		"",
		"          envFrom:",
		"          - configMapRef:",
		fmt.Sprintf("              name: env-%s", repoName),
		"",
	}
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	i := 0
	lines := []byte{}
	for scanner.Scan() {
		line := scanner.Text()
		if i < len(toRemove) {
			if strings.Contains(line, toRemove[i]) {
				line = toReplace[i]
				i += 1
			}
		}
		if i == len(toRemove) && strings.Contains(line, "envFrom") {
			continue
		}
		if strings.Contains(line, "{{ secrets_reader_config_map }}") {
			// See `ansible-deployers/vars/main.yml`.
			//			line = strings.Replace(line, "{{ secrets_reader_config_map }}", "kubernetes-container-user", 1)
			line = strings.Replace(line, "{{ secrets_reader_config_map }}", "kubernetes-container-user", 1)
		}
		// At least one deployment has tabs that confuse the yaml parser and throws an exception.
		// Fix it by capturing the text and adding the appropriate number of spaces (10)...we hope...
		// JARNTUY: Just Another Reason Not To Use YAML
		if reImagePullPolicy.MatchString(line) {
			matches := reImagePullPolicy.FindStringSubmatch(line)
			line = fmt.Sprintf("          %s", matches[1])
		}
		lines = fmt.Appendln(lines, line)
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}
	return lines
}

func hackAndBeHappyIngress(contents, env string, hasAdditionalCerts bool) []string {
	inspected := []string{}
	removing := false
	i := 0
	for _, line := range strings.Split(contents, "\n") {
		if (removing && i < 5) || strings.Contains(line, "if additional_cert") {
			removing = true
			if hasAdditionalCerts && i == 1 {
				inspected = append(inspected, line)
			} else if !hasAdditionalCerts && i == 3 {
				inspected = append(inspected, line)
			}
			i += 1
			continue
		}
		//		if strings.Contains(line, "{{ access_subnet_ids }}") {
		//			line = strings.Replace(line, "{{ access_subnet_ids }}", "TODO-access_subnet_ids-TODO", 1)
		//		}
		inspected = append(inspected, line)
	}

	final := []string{}
	var keepLines bool
	if slices.Contains([]string{"development", "beta", "production"}, env) {
		keepLines = true
	}
	for _, line := range inspected {
		if keepLines && strings.Contains(line, "{%") {
			continue
			// TODO
			//		} else if !keepLines && strings.Contains(line, "{%") {
			//			for !strings.Contains(line, "{% endif %}") {
			//				continue
			//			}
			//
		}
		final = append(final, line)
	}
	//		{% if application_environment in ['development', 'beta', 'production'] %}
	//	for _, line := range inspected {
	//		fmt.Println("line", line)
	//	}
	return final
}
