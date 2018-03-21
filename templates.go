package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

var (
	wsRegex   = regexp.MustCompile(`\s+`)
	slugRegex = regexp.MustCompile(`[^a-zA-Z0-9\-\s]`)

	funcMap = template.FuncMap{
		"readmore":  readmore,
		"slugify":   slugify,
		"sortItems": sortItems,
	}
)

func readmore(body, link string) string {
	lineLimit := 5

	lines := strings.Split(body, "\n")
	if len(lines) <= lineLimit {
		return body
	}

	return fmt.Sprintf("%s\n<a class=\"read-more\" href=\"%s\">read more</a>", strings.Join(lines[:lineLimit], "\n"), link)
}

func slugify(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = wsRegex.ReplaceAllString(slug, "-")
	slug = slugRegex.ReplaceAllString(slug, "")
	return slug
}

func sortItems(key string, items []*file) []*file {
	var rev bool
	if key[0] == '-' {
		key = key[1:]
		rev = true
	}
	sorted := make([]*file, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		return rev != (sorted[i].Metadata[key] < sorted[j].Metadata[key])
	})
	return sorted
}

func loadTemplates() (map[string]*template.Template, error) {
	t := make(map[string]*template.Template)

	includes, err := filepath.Glob(filepath.Join(*templates, "include", "*.html"))
	if err != nil {
		return nil, err
	}

	templates, err := filepath.Glob(filepath.Join(*templates, "*.html"))
	if err != nil {
		return nil, err
	}

	mt := template.New("main").Funcs(funcMap)

	t["main"] = mt

	mt, err = mt.Parse(`{{define "main" }} {{ block "base" . }} {{ . }} {{ end }} {{ end }}`)
	if err != nil {
		return nil, err
	}
	for _, file := range templates {
		n := filepath.Base(file)
		files := append(includes, file)
		tmp, err := mt.Clone()
		if err != nil {
			return nil, err
		}
		t[n], err = tmp.ParseFiles(files...)
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}
