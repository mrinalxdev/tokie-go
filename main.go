package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"
)

type LanguageStats struct {
	FileCount int
	LineCount int
	ByteCount int64
	mutex     sync.Mutex
}

type FileResult struct {
	path     string
	language string
}

type SortOption struct {
	Field     string // "files", "lines", "size"
	Direction string // "asc", "desc"
}

var languageExtMap = map[string]string{
	".go":    "Go",
	".py":    "Python",
	".js":    "JavaScript",
	".ts":    "TypeScript",
	".java":  "Java",
	".cpp":   "C++",
	".c":     "C",
	".rb":    "Ruby",
	".php":   "PHP",
	".rs":    "Rust",
	".swift": "Swift",
	".kt":    "Kotlin",
}

type LanguageData struct {
	Name      string
	Stats     LanguageStats
}

func main() {
	startTime := time.Now()

	excludePtr := flag.String("exclude", "", "Comma-separated list of file patterns to exclude (e.g. '*.json,*.yml')")
	sortPtr := flag.String("sort", "", "Sort by: files/lines/size asc/desc (e.g. 'files desc')")
	skipNodeModules := flag.Bool("skip-node-modules", false, "Skip node_modules directories")
	flag.Parse()

	// Parse sorting options
	var sortOpt SortOption
	if *sortPtr != "" {
		parts := strings.Fields(*sortPtr)
		if len(parts) == 2 {
			sortOpt.Field = strings.ToLower(parts[0])
			sortOpt.Direction = strings.ToLower(parts[1])
		} else {
			fmt.Println("Invalid sort format. Using default sorting.")
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}
	desktopPath := filepath.Join(homeDir, "Desktop")
	excludePatterns := strings.Split(*excludePtr, ",")
	if *excludePtr == "" {
		excludePatterns = nil
	}


	stats := make(map[string]*LanguageStats)
	var statsMutex sync.Mutex

	// channels for the pipeline
	filesChan := make(chan FileResult, 1000)
	done := make(chan bool)

	numWorkers := runtime.NumCPU()

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for result := range filesChan {
				processFile(result.path, result.language, stats, &statsMutex)
			}
		}()
	}

	go func() {
		err := filepath.Walk(desktopPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip node_modules directories if flag is set
			if *skipNodeModules && info.IsDir() && info.Name() == "node_modules" {
				return filepath.SkipDir
			}

			if info.IsDir() {
				return nil
			}

			// Checking exclude patterns
			for _, pattern := range excludePatterns {
				matched, err := filepath.Match(strings.TrimSpace(pattern), filepath.Base(path))
				if err != nil || matched {
					return nil
				}
			}

			ext := strings.ToLower(filepath.Ext(path))
			if lang, ok := languageExtMap[ext]; ok {
				filesChan <- FileResult{path: path, language: lang}
			}
			return nil
		})

		if err != nil {
			fmt.Printf("Error walking directory: %v\n", err)
		}

		close(filesChan)
	}()

	wg.Wait()
	close(done)


	
	languageData := make([]LanguageData, 0, len(stats))
	for lang, stat := range stats {
		languageData = append(languageData, LanguageData{
			Name:  lang,
			Stats: *stat,
		})
	}

	sortLanguageData(languageData, sortOpt)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintf(w, "\n🔍 Code Statistics Report (Desktop Scan)\n\n")
	fmt.Fprintf(w, "Language\tFiles\tLines\tSize (KB)\t\n")
	fmt.Fprintf(w, "--------\t-----\t-----\t---------\t\n")

	totalFiles := 0
	totalLines := 0
	totalSize := int64(0)

	for _, data := range languageData {
		totalFiles += data.Stats.FileCount
		totalLines += data.Stats.LineCount
		totalSize += data.Stats.ByteCount
		fmt.Fprintf(w, "%s\t%d\t%d\t%.2f\t\n",
			data.Name,
			data.Stats.FileCount,
			data.Stats.LineCount,
			float64(data.Stats.ByteCount)/1024,
		)
	}

	fmt.Fprintf(w, "--------\t-----\t-----\t---------\t\n")
	fmt.Fprintf(w, "Total\t%d\t%d\t%.2f\t\n",
		totalFiles,
		totalLines,
		float64(totalSize)/1024,
	)
	w.Flush()

	// Print execution time and configuration
	fmt.Printf("\n⚡ Execution Time: %.2f seconds\n", time.Since(startTime).Seconds())
	if sortOpt.Field != "" {
		fmt.Printf("📊 Sorted by: %s (%s)\n", sortOpt.Field, sortOpt.Direction)
	}
	if *skipNodeModules {
		fmt.Printf("🚫 Excluded node_modules directories\n")
	}
	if len(excludePatterns) > 0 {
		fmt.Println("\n🚫 Excluded Patterns:")
		for _, pattern := range excludePatterns {
			if pattern != "" {
				fmt.Printf("   • %s\n", strings.TrimSpace(pattern))
			}
		}
	}
}

func processFile(path, language string, stats map[string]*LanguageStats, statsMutex *sync.Mutex) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	statsMutex.Lock()
	if _, exists := stats[language]; !exists {
		stats[language] = &LanguageStats{}
	}
	stats[language].FileCount++
	stats[language].LineCount += lineCount
	stats[language].ByteCount += info.Size()
	statsMutex.Unlock()
}

func sortLanguageData(data []LanguageData, opt SortOption) {
	sort.Slice(data, func(i, j int) bool {
		var comparison bool
		switch opt.Field {
		case "files":
			comparison = data[i].Stats.FileCount < data[j].Stats.FileCount
		case "lines":
			comparison = data[i].Stats.LineCount < data[j].Stats.LineCount
		case "size":
			comparison = data[i].Stats.ByteCount < data[j].Stats.ByteCount
		default:
			// Default sort by language name
			comparison = data[i].Name < data[j].Name
		}

		// Reverse for descending order
		if opt.Direction == "desc" {
			comparison = !comparison
		}
		return comparison
	})
}