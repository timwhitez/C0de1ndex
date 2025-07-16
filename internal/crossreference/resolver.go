package crossreference

import (
	"C0de1ndex/internal/types"
	"strings"
)

// ResolveCrossReferences populates the ImportedBy and CalledBy fields.
func ResolveCrossReferences(analyses []*types.FileAnalysis) {
	// Create a map for quick lookup of file paths to their analysis.
	analysisMap := make(map[string]*types.FileAnalysis)
	for _, analysis := range analyses {
		analysisMap[analysis.FilePath] = analysis
	}

	// Iterate through each file analysis to resolve references.
	for _, analysis := range analyses {
		// Resolve file-level imports.
		for _, importedFile := range analysis.Dependencies {
			if targetAnalysis, ok := analysisMap[importedFile]; ok {
				// Avoid duplicates
				found := false
				for _, existingImporter := range targetAnalysis.ImportedBy {
					if existingImporter == analysis.FilePath {
						found = true
						break
					}
				}
				if !found {
					targetAnalysis.ImportedBy = append(targetAnalysis.ImportedBy, analysis.FilePath)
				}
			}
		}

		// Resolve function calls.
		for i := range analysis.Functions {
			for _, call := range analysis.Functions[i].CallsTo {
				// This is a simplified example. A real implementation would need
				// to parse the call string to identify the file and function name.
				var filePath, functionName string
				parts := strings.Split(call, ":")
				if len(parts) == 2 {
					filePath = parts[0]
					functionName = parts[1]
				} else if len(parts) == 1 {
					// Assume it's a call to a function in the same file or a global function
					filePath = analysis.FilePath
					functionName = parts[0]
				} else {
					continue // Skip malformed call strings
				}

				// Search for the called function in the specified file or globally
				if targetAnalysis, ok := analysisMap[filePath]; ok {
					for j := range targetAnalysis.Functions {
						if targetAnalysis.Functions[j].Name == functionName {
							// To avoid duplicates
							found := false
							for _, existingCaller := range targetAnalysis.Functions[j].CalledBy {
								if existingCaller == analysis.Functions[i].Name {
									found = true
									break
								}
							}
							if !found {
								targetAnalysis.Functions[j].CalledBy = append(targetAnalysis.Functions[j].CalledBy, analysis.Functions[i].Name)
							}
						}
					}
				}
			}
		}
	}
}