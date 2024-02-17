package main

import (
	"html/template"
	"strings"
)

func getTemplateFuncMap() template.FuncMap {
	templateFuncMap := template.FuncMap{
		"StringsJoin": strings.Join,
	}

	return templateFuncMap
}
