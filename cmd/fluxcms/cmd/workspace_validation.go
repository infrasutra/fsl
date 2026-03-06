package cmd

import (
	"sort"

	"github.com/infrasutra/fsl/parser"
)

func parseFilesWithWorkspaceTypes(fileContents map[string]string) map[string]*parser.DiagnosticsResult {
	allTypes := make(map[string]struct{})
	fileTypes := make(map[string]map[string]struct{}, len(fileContents))

	for file, content := range fileContents {
		result := parser.ParseWithDiagnostics(content)
		current := make(map[string]struct{})
		if result.Schema != nil {
			for _, typeDef := range result.Schema.Types {
				current[typeDef.Name] = struct{}{}
				allTypes[typeDef.Name] = struct{}{}
			}
		}
		fileTypes[file] = current
	}

	results := make(map[string]*parser.DiagnosticsResult, len(fileContents))
	for file, content := range fileContents {
		externalTypes := make([]string, 0, len(allTypes))
		for typeName := range allTypes {
			if _, isLocalType := fileTypes[file][typeName]; isLocalType {
				continue
			}
			externalTypes = append(externalTypes, typeName)
		}
		sort.Strings(externalTypes)

		results[file] = parser.ParseWithDiagnosticsAndExternalTypes(content, externalTypes)
	}

	return results
}
