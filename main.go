package main

import (
	"C0de1ndex/internal/analyzer"
	"C0de1ndex/internal/cache"
	"C0de1ndex/internal/config"
	"C0de1ndex/internal/crossreference"
	"C0de1ndex/internal/llm"
	"C0de1ndex/internal/prompts"
	"C0de1ndex/internal/reporting"
	"C0de1ndex/internal/types"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

func main() {
	// Load configuration from .env file
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Load prompts from file
	promptData, err := prompts.LoadPrompts("prompts.json")
	if err != nil {
		fmt.Printf("Error loading prompts: %v\n", err)
		os.Exit(1)
	}

	// Initialize the LLM client
	llmClient, err := llm.NewLLMClient(cfg, promptData)
	if err != nil {
		fmt.Printf("Error initializing LLM client: %v\n", err)
		os.Exit(1)
	}

	// 1. Define and parse command-line flags
	var dirPath, outputPath, outputFormat, includeExts, excludePatterns string
	var deep, debug bool
	var concurrency int

	flag.StringVar(&dirPath, "t", ".", "要索引的目标目录路径")
	flag.StringVar(&outputPath, "o", "./output", "输出报告的目录")
	flag.StringVar(&outputFormat, "fmt", "md", "输出格式: 'md' (Markdown) 或 'json'")
	flag.BoolVar(&deep, "deep", false, "启用深度分析模式")
	flag.StringVar(&includeExts, "in", "", "要包含的文件扩展名列表 (逗号分隔, 例如: .go,.js)")
	flag.StringVar(&excludePatterns, "ex", "", "要排除的文件/目录的 glob 模式列表 (逗号分隔)")
	flag.IntVar(&concurrency, "c", 2, "并发文件分析的工作程序数")
	flag.BoolVar(&debug, "debug", false, "启用调试模式以获取详细输出")
	flag.Parse()

	// Pass debug flag to LLM client
	llmClient.Debug = debug

	// Determine which model to use
	model := cfg.DefaultModel
	if deep {
		model = cfg.DeepModel
	}

	fmt.Printf("Using model: %s\n", model)

	// Check if the provided path is a valid directory
	if info, err := os.Stat(dirPath); err != nil || !info.IsDir() {
		fmt.Println("Error: Provided path is not a valid directory.")
		os.Exit(1)
	}

	fmt.Printf("开始分析项目: %s\n", dirPath)

	// 2. List all files in the directory
	allFiles, err := analyzer.ListFiles(dirPath)
	if err != nil {
		fmt.Printf("Error listing files: %v\n", err)
		os.Exit(1)
	}

	// 3. Parse the .gitignore file
	ignorePatterns, err := analyzer.ParseGitignore(dirPath)
	if err != nil {
		fmt.Printf("Error reading .gitignore: %v\n", err)
		os.Exit(1)
	}

	// 4. First apply rule-based filtering
	fmt.Println("应用规则过滤...")
	// Parse exclude patterns from command line
	var excludePatternsSlice []string
	if excludePatterns != "" {
		excludePatternsSlice = strings.Split(excludePatterns, ",")
		for i := range excludePatternsSlice {
			excludePatternsSlice[i] = strings.TrimSpace(excludePatternsSlice[i])
		}
	}
	ruleFilteredFiles := analyzer.FilterFilesByRules(allFiles, ignorePatterns, excludePatternsSlice)
	fmt.Printf("规则过滤后剩余 %d 个文件\n", len(ruleFilteredFiles))

	// Apply include extensions filter before LLM filtering
	if includeExts != "" {
		extensions := strings.Split(includeExts, ",")
		var tempFiltered []string
		for _, file := range ruleFilteredFiles {
			for _, ext := range extensions {
				if strings.HasSuffix(file, strings.TrimSpace(ext)) {
					tempFiltered = append(tempFiltered, file)
					break
				}
			}
		}
		ruleFilteredFiles = tempFiltered
		fmt.Printf("扩展名过滤后剩余 %d 个文件\n", len(ruleFilteredFiles))
	}

	// 5. Then apply LLM filtering
	fmt.Println("使用LLM进行智能过滤...")
	filteredFiles, err := llmClient.FilterFilesWithLLM(ruleFilteredFiles, ignorePatterns, model)
	if err != nil {
		fmt.Printf("Error filtering files with LLM: %v\n", err)
		os.Exit(1)
	}





	if debug {
		fmt.Println("\n--- Filtered Files (Post-Exclusion) ---")
		for _, file := range filteredFiles {
			fmt.Println(file)
		}
	}

	// Initialize the progress bar
	bar := progressbar.NewOptions(len(filteredFiles),
		progressbar.OptionSetDescription("Analyzing files"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetTheme(progressbar.Theme{Saucer: "=", SaucerHead: ">", SaucerPadding: " ", BarStart: "[", BarEnd: "]"}),
	)

	// Initialize the cache
	if err := cache.InitCache(); err != nil {
		fmt.Printf("Error initializing cache: %v\n", err)
		os.Exit(1)
	}

	// 5. Concurrently analyze the filtered files
	jobs := make(chan string, len(filteredFiles))
	results := make(chan *types.FileAnalysis, len(filteredFiles))

	// Start workers
	for w := 0; w < concurrency; w++ {
		go func(jobs <-chan string, results chan<- *types.FileAnalysis) {
			for file := range jobs {
				func() {
					defer bar.Add(1)
					// Construct absolute path to read the file
					absPath := filepath.Join(dirPath, file)
					content, err := os.ReadFile(absPath)
					if err != nil {
						fmt.Printf("\nError reading file %s: %v\n", file, err)
						results <- nil // Send nil to indicate failure
						return
					}

					hash := cache.GetContentHash(content)
					cached, err := cache.GetCachedAnalysis(hash)
					if err != nil {
						fmt.Printf("\nError reading cache for %s: %v\n", file, err)
					}

					if cached != nil {
						results <- cached
						return
					}

					// (LLM) Analyze each file
					analysis, err := analyzer.AnalyzeFileWithLLM(llmClient, file, string(content), model)
					if err != nil {
						fmt.Printf("\nError analyzing file %s with LLM: %v\n", file, err)
						results <- nil
						return
					}

					// Save the new analysis to the cache
					if err := cache.SaveAnalysisToCache(hash, analysis); err != nil {
						fmt.Printf("\nError saving to cache for %s: %v\n", file, err)
					}

					results <- analysis
				}()
			}
		}(jobs, results)
	}

	// Send jobs to the workers
	for _, file := range filteredFiles {
		jobs <- file
	}
	close(jobs)

	// Collect the results
	var allAnalyses []*types.FileAnalysis
	totalFiles := len(filteredFiles)
	if totalFiles > 0 {
		fmt.Printf("\nAnalyzing %d files...\n", totalFiles)
	}
	for i := 0; i < totalFiles; i++ {
		analysis := <-results
		if analysis != nil {
			allAnalyses = append(allAnalyses, analysis)
		}
	}
	if totalFiles > 0 {
		fmt.Println() // Newline after the progress bar is complete.
	}

	// 6. Resolve cross-references
	if debug {
		fmt.Println("\n--- Raw Analysis Results ---")
		jsonOutput, err := json.MarshalIndent(allAnalyses, "", "  ")
		if err != nil {
			fmt.Printf("Error marshalling raw analysis: %v\n", err)
		} else {
			fmt.Println(string(jsonOutput))
		}
		fmt.Println("--------------------------")
	}
	fmt.Println("正在解析交叉引用...")
	crossreference.ResolveCrossReferences(allAnalyses)

	// 7. Generate Project Summary
	fmt.Println("正在生成项目摘要...")
	projectSummary, err := llmClient.GenerateProjectSummary(allAnalyses, model)
	if err != nil {
		fmt.Printf("Error generating project summary: %v\n", err)
		os.Exit(1)
	}

	// Create the output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// 7. Generate final report
	fmt.Println("正在生成最终报告...")
	timestamp := time.Now().Format("20060102_150405")
	outputFile := filepath.Join(outputPath, fmt.Sprintf("context_%s.%s", timestamp, strings.ToLower(outputFormat)))

	if strings.ToLower(outputFormat) == "json" {
		// Create the full JSON report structure
		jsonReport := &types.JSONReport{
			AnalysisTime:   time.Now().Format("2006-01-02 15:04:05"),
			ProjectSummary: projectSummary,
			DirectoryTree:  filteredFiles,
			FileAnalyses:   allAnalyses,
		}

		jsonContent, err := json.MarshalIndent(jsonReport, "", "  ")
		if err != nil {
			fmt.Printf("Error marshalling to JSON: %v\n", err)
			os.Exit(1)
		}
		if err := reporting.SaveReportToFile(string(jsonContent), outputFile); err != nil {
			fmt.Printf("Error saving JSON report: %v\n", err)
			os.Exit(1)
		}
	} else {
		reportContent, err := reporting.GenerateMarkdownReport(projectSummary, allAnalyses, filteredFiles)
		if err != nil {
			fmt.Printf("Error generating report: %v\n", err)
			os.Exit(1)
		}

		if err := reporting.SaveReportToFile(reportContent, outputFile); err != nil {
			fmt.Printf("Error saving report to file: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("\n项目分析完成。报告已保存到 %s\n", outputFile)
}
