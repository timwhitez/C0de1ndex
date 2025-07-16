package types

// FileAnalysis holds the structured analysis of a single file.
// This struct is designed to be populated by the LLM.
type FileAnalysis struct {
	FilePath     string         `json:"file_path"`
	Summary      string         `json:"summary"`
	Dependencies []string       `json:"dependencies"`
	Imports      []string       `json:"imports"`
	ImportedBy   []string       `json:"imported_by"`
	Functions    []FunctionInfo `json:"functions"`
}

// FunctionInfo holds the analysis of a single function.
type FunctionInfo struct {
	Name        string   `json:"name"`
	Signature   string   `json:"signature"`
	Description string   `json:"description"`
	Parameters  []string `json:"parameters"`
	ReturnValue string   `json:"return_value"`
	CallsTo     []string `json:"calls_to"`
	CalledBy    []string `json:"called_by"`
}

// ProjectSummary holds the high-level analysis of the entire project.
// ProjectSummary 保存了整个项目的高级分析结果。

type CoreModule struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Files       []string `json:"files"`
}

type ProjectSummary struct {
	ProjectSummary  string       `json:"project_summary"`
	TechStack       []string     `json:"tech_stack"`
	CoreModules     []CoreModule `json:"core_modules"`
	DependencyGraph string       `json:"dependency_graph"`
}

// JSONReport represents the full JSON output.
type JSONReport struct {
	AnalysisTime    string          `json:"analysis_time"`
	ProjectSummary  *ProjectSummary `json:"project_summary"`
	DirectoryTree   []string        `json:"directory_tree"`
	FileAnalyses    []*FileAnalysis `json:"file_analyses"`
}
