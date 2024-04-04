package main

import (
	"encoding/base64"
	"html/template"
	"strings"
)

func getTemplateFuncMap() template.FuncMap {
	templateFuncMap := template.FuncMap{
		"StringsJoin": strings.Join,
		"base64":      base64.StdEncoding.EncodeToString,
	}

	return templateFuncMap
}
