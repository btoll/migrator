package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var reToken = regexp.MustCompile(`{{\s*[^{}][a-zA-Z_]*\s*}}`)

func checkFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func getCertificateValues(mergedValues map[string]interface{}) (string, string, error) {
	apexDomain, ok := mergedValues["apex_domain"]
	if ok {
		applicationRegion, ok := mergedValues["application_region"]
		if ok {
			return apexDomain.(string), applicationRegion.(string), nil
		}
	}
	return "", "", errors.New("Unable to retrieve certificate values")
}

func getManifestValues(filename string) ManifestValues {
	base := make(ManifestValues)
	if checkFileExists(filename) {
		b, err := os.ReadFile(filename)
		if err != nil {
			panic(err)
		}
		err = yaml.Unmarshal(b, &base)
		if err != nil {
			panic(err)
		}
	}
	return base
}

func getTokensFromManifest(manifest string) map[string]int {
	tokenMap := make(map[string]int)
	matches := reToken.FindAllString(manifest, -1)
	for _, match := range matches {
		if _, ok := tokenMap[match]; !ok {
			tokenMap[match] = 0
		}
		tokenMap[match] += 1
	}
	return tokenMap
}

func mapMerge(maps ...map[string]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			// TODO
			if k != "environment_variables" {
				res[k] = v
			}
		}
	}
	return res
}

// Order matters. The last item in the slice will override anything/everything previously.
func replaceMerge(envvars map[string]string, vs ...T) {
	for _, v := range vs {
		_struct_EnvironmentVariables := reflect.ValueOf(v)
		_struct_EnvironmentVariable := _struct_EnvironmentVariables.Field(0)
		for i := 0; i < _struct_EnvironmentVariable.Len(); i++ {
			s := _struct_EnvironmentVariable.Index(i)
			key := s.Field(0).Interface()
			val := s.Field(1).Interface()
			envvars[key.(string)] = val.(string)
		}
	}
}

func tokenizeManifests(data map[string]interface{}, manifest string) (string, map[string]int) {
	tokenMap := getTokensFromManifest(manifest)
	templated := manifest
	for k, v := range data {
		// TODO should log these key/values
		var s string
		switch v.(type) {
		case string:
			s = v.(string)
		case int:
			s = strconv.Itoa(v.(int))
		}
		var token string = fmt.Sprintf("{{ %s }}", k)
		if strings.Contains(templated, token) {
			templated = strings.ReplaceAll(templated, token, s)
			tokenMap[token] = 0
		}
	}
	return templated, tokenMap
}

func writeManifestFile(filename, contents, appDir string) {
	f, err := os.Create(fmt.Sprintf("%s/base/%s", appDir, filename))
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	_, err = io.WriteString(f, contents)
	if err != nil {
		fmt.Println(err)
	}
}
