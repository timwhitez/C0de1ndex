package reporting

import (
	"C0de1ndex/internal/types"
	"fmt"
	"os"
	"strings"
	"time"
)

// GenerateMarkdownReport creates a detailed analysis report in Markdown format.
func GenerateMarkdownReport(summary *types.ProjectSummary, analyses []*types.FileAnalysis, fullTree []string) (string, error) {
	var builder strings.Builder

	// --- Header ---
	builder.WriteString("# 项目代码分析文档\n\n")
	builder.WriteString(fmt.Sprintf("分析时间: %s\n\n", time.Now().Format("2006-01-02_15-04-05")))

	// --- Project Summary ---
	if summary != nil {
		builder.WriteString("## 项目全局摘要\n\n")
		builder.WriteString(fmt.Sprintf("**项目概述**: %s\n\n", summary.ProjectSummary))
		builder.WriteString("**核心模块**:\n")
		for _, module := range summary.CoreModules {
			builder.WriteString(fmt.Sprintf("- %s\n", module))
		}
		builder.WriteString("\n**模块依赖图 (Mermaid.js)**:\n")
		builder.WriteString("```mermaid\n")
		builder.WriteString(summary.DependencyGraph)
		builder.WriteString("\n```\n\n")
	}

	// --- Directory Structure (Filtered) ---

	builder.WriteString("## 项目目录结构 (已过滤)\n")
	builder.WriteString("```\n")
	// Note: This is a simplified view of the original tree. A full tree might be too large.
	// We are showing the files that were considered for analysis.
	builder.WriteString(strings.Join(fullTree, "\n"))
	builder.WriteString("\n```\n\n")

	// --- Excluded Files (Placeholder) ---
	builder.WriteString("### 排除的目录和文件\n")
	builder.WriteString("**以下目录/文件已根据规则在分析前被过滤:**\n")
	builder.WriteString("- .gitignore, node_modules, venv, build, target, etc.\n\n")

	// --- File Analysis Section ---
	builder.WriteString("## 文件分析详情\n")

	if len(analyses) == 0 {
		builder.WriteString("未分析任何文件。\n")
	} else {
		for _, analysis := range analyses {
			builder.WriteString(fmt.Sprintf("\n---\n\n### 文件: `%s`\n\n", analysis.FilePath))

			// File Summary
			builder.WriteString("#### 文件概述\n")
			builder.WriteString(fmt.Sprintf("%s\n\n", analysis.Summary))

			// Dependencies
			builder.WriteString("#### 依赖关系\n")
			if len(analysis.Dependencies) > 0 {
				builder.WriteString(fmt.Sprintf("- **导入**: `%s`\n", strings.Join(analysis.Dependencies, "`, `")))
			} else {
				builder.WriteString("- **导入**: 无\n")
			}
			if len(analysis.ImportedBy) > 0 {
				builder.WriteString(fmt.Sprintf("- **被导入**: `%s`\n\n", strings.Join(analysis.ImportedBy, "`, `")))
			} else {
				builder.WriteString("- **被导入**: 无\n\n")
			}

			// Functions List
			builder.WriteString("#### 函数列表\n")
			if len(analysis.Functions) == 0 {
				builder.WriteString("未在此文件中检测到主要函数。\n")
			} else {
				for _, function := range analysis.Functions {
					builder.WriteString(fmt.Sprintf("##### 函数: `%s`\n", function.Name))
					builder.WriteString(fmt.Sprintf("- **签名**: `%s`\n", function.Signature))
					builder.WriteString(fmt.Sprintf("- **描述**: %s\n", function.Description))

					// Parameters
					if len(function.Parameters) > 0 {
						builder.WriteString("- **参数**:\n")
						for _, param := range function.Parameters {
							builder.WriteString(fmt.Sprintf("  - `%s`\n", param))
						}
					} else {
						builder.WriteString("- **参数**: 无\n")
					}

					// Return Value
					builder.WriteString(fmt.Sprintf("- **返回**: %s\n", function.ReturnValue))

					// Call Relationships
					builder.WriteString("- **调用关系**:\n")
					if len(function.CallsTo) > 0 {
						builder.WriteString(fmt.Sprintf("  - **调用**: `%s`\n", strings.Join(function.CallsTo, "`, `")))
					} else {
						builder.WriteString("  - **调用**: 无\n")
					}
					if len(function.CalledBy) > 0 {
						builder.WriteString(fmt.Sprintf("  - **被调用**: `%s`\n\n", strings.Join(function.CalledBy, "`, `")))
					} else {
						builder.WriteString("  - **被调用**: 无\n\n")
					}
				}
			}
		}
	}

	return builder.String(), nil
}

// SaveReportToFile writes the final report to a file.
func SaveReportToFile(content string, outputPath string) error {
	return os.WriteFile(outputPath, []byte(content), 0644)
}
