package cmd

import (
	"sort"

	"github.com/infrasutra/fsl/parser"
)

func workspaceOptions(file string, fileSchemas map[string]*parser.Schema) parser.Options {
	local := map[string]bool{}
	if schema := fileSchemas[file]; schema != nil {
		for _, typeDef := range schema.Types {
			local[typeDef.Name] = true
		}
	}
	external := map[string]bool{}
	library := &parser.Schema{}
	for other, schema := range fileSchemas {
		if other == file || schema == nil {
			continue
		}
		library.Types = append(library.Types, schema.Types...)
		library.Enums = append(library.Enums, schema.Enums...)
		for _, typeDef := range schema.Types {
			if !local[typeDef.Name] {
				external[typeDef.Name] = true
			}
		}
	}
	names := make([]string, 0, len(external))
	for name := range external {
		names = append(names, name)
	}
	sort.Strings(names)
	sort.SliceStable(library.Types, func(i, j int) bool { return library.Types[i].Name < library.Types[j].Name })
	sort.SliceStable(library.Enums, func(i, j int) bool { return library.Enums[i].Name < library.Enums[j].Name })
	return parser.Options{ExternalTypes: names, Library: library}
}

func parseFilesWithWorkspaceTypes(fileContents map[string]string) (map[string]*parser.DiagnosticsResult, map[string]parser.Options) {
	fileSchemas := make(map[string]*parser.Schema, len(fileContents))
	for file, content := range fileContents {
		fileSchemas[file] = parser.ParseWithDiagnostics(content).Schema
	}

	results := make(map[string]*parser.DiagnosticsResult, len(fileContents))
	options := make(map[string]parser.Options, len(fileContents))
	for file, content := range fileContents {
		options[file] = workspaceOptions(file, fileSchemas)
		results[file] = parser.ParseWithDiagnosticsAndOptions(content, options[file])
	}

	return results, options
}
