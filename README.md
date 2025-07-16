# C0de1ndex: AI 驱动的代码库索引工具

`C0de1ndex` 是一个命令行工具，它使用大型语言模型 (LLM) 来智能地分析和创建软件项目的综合索引。它会遍历代码库，使用 `.gitignore` 和标准排除规则过滤掉不相关的文件，并为每个文件的用途、功能和依赖关系生成详细的报告。

该工具旨在帮助开发人员快速理解新的或复杂的代码库，自动化文档，并促进代码审查。

## 功能

- **AI 驱动的分析**: 利用 LLM 分析代码并生成人类可读的摘要。
- **智能过滤**: 根据 `.gitignore` 和内置规则，自动忽略依赖项、构建产物和其他非必要文件。
- **可配置**: 通过各种命令行标志来控制分析。
- **多种输出格式**: 生成 Markdown (`.md`) 或 JSON (`.json`) 格式的报告。
- **可定制的提示**: 发送给 LLM 的所有提示都存储在 `prompts.json` 中，允许高级用户根据自己的需求定制分析。
- **语言感知**: 自动检测每个文件的编程语言，以提高分析质量。
- **并发分析**: 支持并发文件分析，以加快处理速度。
- **强大的缓存机制**: 缓存已分析文件的结果，以避免对未更改的文件进行不必要的 API 调用。
- **增强的重试逻辑**: 具有两阶段重试机制，以处理临时的 API 或网络问题。

## 设置

在运行该工具之前，您需要提供您的 OpenAI API（或其他兼容的）凭据。

1.  **创建 `.env` 文件**: 将示例配置文件复制到一个名为 `.env` 的新文件中。

    ```bash
    cp .env.example .env
    ```

2.  **编辑 `.env`**: 打开 `.env` 文件，并添加您的 API 密钥和 API 的基本 URL。

    ```dotenv
    # Your OpenAI API Key
    OPENAI_API_KEY="sk-your_api_key_here"

    # The base URL for the OpenAI API (or a compatible proxy)
    OPENAI_BASE_URL="https.api.openai.com/v1"

    # The default model for standard analysis
    DEFAULT_MODEL="your_default_model_here"

    # The model for deep analysis (used when the --deep flag is set)
    DEEP_MODEL="your_deep_model_here"
    ```

## 用法

该工具从命令行运行，使用 `go run .`。它接受几个标志来自定义其行为。

```
  -c int
        并发文件分析的工作程序数 (default 2)
  -debug
        启用调试模式以获取详细输出
  -deep
        启用深度分析模式
  -ex string
        要排除的文件/目录的 glob 模式列表 (逗号分隔)
  -fmt string
        输出格式: 'md' (Markdown) 或 'json' (default "md")
  -in string
        要包含的文件扩展名列表 (逗号分隔, 例如: .go,.js)
  -o string
        输出报告的目录 (default "./output")
  -t string
        要索引的目标目录路径 (default ".")
```

### 基本用法

要分析当前目录并生成 `context_<timestamp>.md` 报告：

```bash
go run .
```

### 高级用法

- **`-t`**: 您要分析的代码目录的路径。
  ```bash
  go run . -t /path/to/your/project
  ```

- **`-o`**: 输出文件的基本名称 (扩展名将自动添加)。
  ```bash
  go run . -o my_project_analysis
  ```

- **`-fmt`**: 输出格式。可以是 `md` (默认) 或 `json`。
  ```bash
  go run . -fmt json
  ```

- **`-deep`**: 启用深度分析模式，使用指定的 `DEEP_MODEL`。
  ```bash
  go run . -deep
  ```

- **`-in`**: 要包含的文件扩展名的逗号分隔列表 (例如, .go,.js,.py)。
  ```bash
  go run . -in .go,.js
  ```

- **`-ex`**: 要明确排除的文件/目录的 glob 模式的逗号分隔列表。
  ```bash
  go run . -ex "*.test.go,docs/*,*.xml"
  ```

- **`-c`**: 并发文件分析工作程序的数量。
  ```bash
  go run . -c 4
  ```

- **`-debug`**: 启用调试模式以获取详细输出。
  ```bash
  go run . -debug
  ```

**所有标志的示例：**

```bash
go run . -t ./my-app -o my-app-report -fmt json -deep -c 8 -ex "*mock*.go"
```

## 输出格式

该工具会生成一份详细的报告，其中包括：

- **项目全局摘要**: 对整个项目的高层次概述，包括其核心模块和依赖关系图。
- **简化的目录树**: 已分析文件的目录树。
- **文件分析详情**: 对每个文件的逐一分解，包括：
  - 文件用途的摘要。
  - 其依赖项的列表。
  - 其功能的详细列表，包括其签名、描述、参数和返回值。

这在单个可搜索的文档中提供了整个代码库的强大的高层次概述。

## 工作原理

### 分析流程

1.  **配置加载**: 从 `.env` 文件加载 `OPENAI_API_KEY` 和模型配置。
2.  **文件扫描**: 递归扫描目标目录，并根据 `.gitignore` 和内置规则进行初步过滤。
3.  **LLM 过滤**: 使用 `filter_prompt`，让 LLM 对文件列表进行智能过滤，移除不相关的代码。
4.  **并发分析**: 并发地处理每个文件：
    a.  **缓存检查**: 检查文件的哈希值是否已存在于缓存中。如果存在，则直接使用缓存的结果。
    b.  **代码分块**: 如果文件过大，会将其分割成多个块。
    c.  **LLM 分析**: 使用 `analysis_prompt` 或 `chunk_analysis_prompt`，将文件内容发送给 LLM 进行分析。
    d.  **JSON 修复**: LLM 返回的 JSON 可能会有轻微的格式错误。工具会自动尝试修复这些错误。
    e.  **缓存写入**: 将新的分析结果存入缓存。
5.  **项目总结**: 将所有文件的摘要发送给 LLM，使用 `project_summary_prompt` 生成最终的项目报告。
6.  **报告生成**: 根据指定的格式（Markdown 或 JSON）生成最终的报告文件。

### 缓存机制

为了优化性能和降低成本，`C0de1ndex` 实现了一套强大的缓存机制。

-   **缓存目录**: 在项目根目录下创建一个名为 `.c0de1ndex_cache` 的目录，用于存放缓存文件。
-   **内容哈希**: 在分析文件之前，工具会计算其内容的 SHA-256 哈希值。这个哈希值作为文件状态的唯一标识符。
-   **缓存查找**: 工具会检查缓存目录中是否存在一个名为 `<hash>.json` 的 JSON 文件。
    -   如果文件存在（**缓存命中**），则读取其内容作为分析结果，从而跳过昂贵的 LLM API 调用。
    -   如果文件不存在（**缓存未命中**），则该文件将被 LLM 分析。
-   **缓存存储**: 分析成功后，结果将以内容哈希命名，并保存为 JSON 文件存入缓存目录。这确保了只要文件内容不变，即使文件名或路径改变，其分析结果也能从缓存中获取。

这种基于内容的缓存策略确保了只有在文件被修改后才需要重新分析，从而显著加快了在同一代码库上后续运行的速度。

### 提示工程 (`prompts.json`)

`C0de1ndex` 的核心是其提示工程。`prompts.json` 文件包含了所有与大语言模型 (LLM) 交互的模板。通过修改此文件，您可以精确控制分析的深度、格式和风格。

-   **`filter_prompt`**: 定义了如何指示 LLM 根据 `.gitignore` 和其他规则过滤文件。
-   **`analysis_prompt`**: 这是最重要的提示，它指导 LLM 如何分析单个代码文件，要求其以特定的 JSON 格式返回摘要、依赖项和函数分析。
-   **`project_summary_prompt`**: 指导 LLM 如何将所有单个文件的分析融合成一个连贯的项目摘要。
-   **`chunk_analysis_prompt`**: 用于处理大文件，指导 LLM 分析代码块。

### 项目结构

```
/
├── .c0de1ndex_cache/   # 缓存分析结果
├── internal/           # 项目内部逻辑
│   ├── analyzer/       # 代码分析核心模块
│   ├── cache/          # 缓存管理
│   ├── config/         # 配置加载
│   ├── llm/            # 与大语言模型交互
│   ├── prompts/        # LLM 提示管理
│   ├── reporting/      # 报告生成
│   └── types/          # 全局类型定义
├── .env                # 环境变量 (本地)
├── .env.example        # 环境变量示例
├── go.mod              # Go 模块依赖
├── main.go             # 程序入口
└── prompts.json        # LLM 提示模板
```