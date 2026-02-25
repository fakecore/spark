package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// FieldInfo represents a field with its number and line content
type FieldInfo struct {
	Number int
	Lines  []string // 支持多行字段（包括注释）
}

func main() {
	var protoPath string
	var recursive bool
	var fillGaps bool
	flag.StringVar(&protoPath, "path", "", "Proto file or directory path")
	flag.BoolVar(&recursive, "recursive", false, "Process directories recursively")
	flag.BoolVar(&fillGaps, "fill-gaps", false, "Re-number fields to fill gaps in sequence")
	flag.Parse()

	if protoPath == "" {
		fmt.Println("Usage: go run sort_proto_fields.go -path <proto_file_or_directory_path> [-recursive] [-fill-gaps]")
		os.Exit(1)
	}

	fmt.Printf("Processing path: %s\n", protoPath)

	stat, err := os.Stat(protoPath)
	if err != nil {
		fmt.Printf("Error accessing path: %v\n", err)
		os.Exit(1)
	}

	if stat.IsDir() {
		fmt.Printf("Processing directory: %s (recursive: %v, fill-gaps: %v)\n", protoPath, recursive, fillGaps)
		if err := processDirectory(protoPath, recursive, fillGaps); err != nil {
			fmt.Printf("Error processing directory: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("Processing file: %s (fill-gaps: %v)\n", protoPath, fillGaps)
		if err := sortProtoFields(protoPath, fillGaps); err != nil {
			fmt.Printf("Error sorting proto fields: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully sorted fields in %s\n", protoPath)
	}
}

func processDirectory(dirPath string, recursive bool, fillGaps bool) error {
	processedCount := 0
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not recursive and not in the root directory
		if !recursive && path != dirPath && info.IsDir() {
			return filepath.SkipDir
		}

		// Process only .proto files
		if !info.IsDir() && strings.HasSuffix(path, ".proto") {
			fmt.Printf("Processing proto file: %s\n", path)
			if err := sortProtoFields(path, fillGaps); err != nil {
				fmt.Printf("Error processing %s: %v\n", path, err)
				return err
			}
			processedCount++
			fmt.Printf("Successfully sorted fields in %s\n", path)
		}

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("Total processed files: %d\n", processedCount)
	return nil
}

func sortProtoFields(filePath string, fillGaps bool) error {
	// Read the file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read all content
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Process the content
	content := strings.Join(lines, "\n")
	reorderedContent := reorderProtoMessages(content, fillGaps)

	// Write back to file
	if err := os.WriteFile(filePath, []byte(reorderedContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func reorderProtoMessages(protoContent string, fillGaps bool) string {
	// Regular expression to extract field number from lines
	// 改进正则表达式以更好地处理带验证规则的字段
	fieldNumberRegex := regexp.MustCompile(`=\s*(\d+)\s*(?:\[.*?\])?\s*;`)

	var finalOutput []string
	var messageStack []map[string]interface{}

	lines := strings.Split(protoContent, "\n")

	// 用于跟踪注释和字段的关联
	var currentFieldLines []string
	// 用于跟踪多行注释
	var inMultilineComment bool

	for _, line := range lines {
		strippedLine := strings.TrimSpace(line)

		// 检查是否进入或退出多行注释
		if strings.Contains(line, "/*") && !strings.Contains(line, "*/") {
			inMultilineComment = true
		}
		if strings.Contains(line, "*/") && !strings.Contains(line, "/*") {
			inMultilineComment = false
		}

		// 1. Discover new message definition
		if strings.HasPrefix(strippedLine, "message") && strings.HasSuffix(strippedLine, "{") {
			// 如果有未处理的字段行，添加到上一个message的other_lines中
			if len(currentFieldLines) > 0 && len(messageStack) > 0 {
				currentContext := messageStack[len(messageStack)-1]
				otherLines := currentContext["other_lines"].([]string)
				otherLines = append(otherLines, currentFieldLines...)
				currentContext["other_lines"] = otherLines
				currentFieldLines = []string{}
			}

			indentStr := strings.Repeat("  ", len(messageStack))

			// Create new message context and push to stack
			newMessageContext := map[string]interface{}{
				"header":      line,
				"fields":      []FieldInfo{},
				"other_lines": []string{},
				"indent_str":  indentStr,
			}
			messageStack = append(messageStack, newMessageContext)
			continue
		}

		// 2. Discover message end
		if strings.HasPrefix(strippedLine, "}") && len(messageStack) > 0 {
			// 如果有未处理的字段行，添加到当前message的fields中
			if len(currentFieldLines) > 0 {
				currentContext := messageStack[len(messageStack)-1]
				// 检查是否是字段行
				isField := false
				for _, fieldLine := range currentFieldLines {
					if fieldNumberRegex.MatchString(fieldLine) {
						isField = true
						break
					}
				}

				if isField {
					// 是字段，添加到字段列表
					match := fieldNumberRegex.FindStringSubmatch(strings.Join(currentFieldLines, "\n"))
					if match != nil {
						fieldNumber := parseInt(match[1])
						fields := currentContext["fields"].([]FieldInfo)
						fields = append(fields, FieldInfo{
							Number: fieldNumber,
							Lines:  currentFieldLines,
						})
						currentContext["fields"] = fields
					}
					// 清空当前字段行
					currentFieldLines = []string{}
				} else {
					// 不是字段，添加到其他行
					otherLines := currentContext["other_lines"].([]string)
					otherLines = append(otherLines, currentFieldLines...)
					currentContext["other_lines"] = otherLines
				}
				currentFieldLines = []string{}
			}

			// a. Pop current message from stack
			currentMessage := messageStack[len(messageStack)-1]
			messageStack = messageStack[:len(messageStack)-1]

			// b. Sort fields by number
			fields := currentMessage["fields"].([]FieldInfo)
			sort.Slice(fields, func(i, j int) bool {
				return fields[i].Number < fields[j].Number
			})

			// c. 检查字段编号是否连续或是否有重复，如果不连续或有重复则重新编号
			// 如果fillGaps为true，也重新编号以填补断层
			if !isContinuousSequence(fields) || hasDuplicateFieldNumbers(fields) || fillGaps {
				usedNumbers := make(map[int]bool)
				nextAvailableNumber := 1
				for i := range fields {
					originalNumber := fields[i].Number
					// 如果fillGaps为true，则始终重新编号；否则仅在当前编号已被使用时重新编号
					newNumber := originalNumber
					if fillGaps {
						// 当fillGaps为true时，始终分配下一个可用的连续编号
						for usedNumbers[nextAvailableNumber] {
							nextAvailableNumber++
						}
						newNumber = nextAvailableNumber
						nextAvailableNumber++
					} else if usedNumbers[originalNumber] {
						// 只有在有重复时才分配下一个可用编号
						// 找到下一个可用的连续编号
						for usedNumbers[nextAvailableNumber] {
							nextAvailableNumber++
						}
						newNumber = nextAvailableNumber
						nextAvailableNumber++
					}
					usedNumbers[newNumber] = true
					fields[i].Number = newNumber
					// 更新字段行中的编号
					for j, line := range fields[i].Lines {
						// 使用正则表达式替换字段编号，保留验证规则
						re := regexp.MustCompile(fmt.Sprintf(`=\s*%d(\s*(?:\[.*?\])?\s*;)`, originalNumber))
						fields[i].Lines[j] = re.ReplaceAllString(line, fmt.Sprintf("= %d$1", newNumber))
					}
				}
				// 重新排序字段
				sort.Slice(fields, func(i, j int) bool {
					return fields[i].Number < fields[j].Number
				})
			}

			// d. Prepare formatted output
			parentIndentStr := currentMessage["indent_str"].(string)
			innerIndentStr := parentIndentStr + "  "

			// Assemble message block
			var formattedBlock []string
			formattedBlock = append(formattedBlock, currentMessage["header"].(string))

			// Add sorted fields
			for _, field := range fields {
				for _, fieldLine := range field.Lines {
					formattedBlock = append(formattedBlock, innerIndentStr+strings.TrimLeft(fieldLine, " \t"))
				}
			}

			// Add non-field lines (like comments and processed nested messages)
			// These should be added at the end, after all fields
			otherLines := currentMessage["other_lines"].([]string)
			for _, otherLine := range otherLines {
				// If otherLine is a processed block, it already has its own indentation
				if strings.Contains(otherLine, "\n") {
					formattedBlock = append(formattedBlock, otherLine)
				} else {
					// Otherwise, add indentation for single line comments etc.
					formattedBlock = append(formattedBlock, innerIndentStr+strings.TrimLeft(otherLine, " \t"))
				}
			}

			// Add closing brace
			formattedBlock = append(formattedBlock, parentIndentStr+"}")

			fullBlockStr := strings.Join(formattedBlock, "\n")

			// d. Add processed block to parent or final output
			if len(messageStack) > 0 {
				// If it's a nested message, add the entire block to parent's other_lines
				messageStack[len(messageStack)-1]["other_lines"] = append(
					messageStack[len(messageStack)-1]["other_lines"].([]string),
					fullBlockStr,
				)
			} else {
				// If it's a top-level message, add to final output
				finalOutput = append(finalOutput, fullBlockStr)
			}
			continue
		}

		// 3. Process various lines inside a message
		if len(messageStack) > 0 {
			currentContext := messageStack[len(messageStack)-1]

			// 将行添加到当前字段行中
			currentFieldLines = append(currentFieldLines, line)

			// 如果在多行注释中，继续累积行直到注释结束
			if inMultilineComment {
				// 检查是否是多行注释的结束
				if strings.Contains(line, "*/") {
					inMultilineComment = false
					// 多行注释结束，将整个注释块添加到other_lines
					otherLines := currentContext["other_lines"].([]string)
					otherLines = append(otherLines, currentFieldLines...)
					currentContext["other_lines"] = otherLines
					currentFieldLines = []string{}
				}
				continue
			}

			// 如果是字段行结尾（包含分号）或者空行，则处理字段
			// 检查当前行是否包含分号（可能是字段行）
			containsSemicolon := strings.Contains(strings.TrimSpace(line), ";")
			if containsSemicolon || strings.TrimSpace(line) == "" {
				// 检查是否包含字段定义
				isField := false
				fieldNumber := 0
				// 将所有当前字段行连接成一个字符串，以便处理跨多行的字段定义
				fullFieldContent := strings.Join(currentFieldLines, " ")
				match := fieldNumberRegex.FindStringSubmatch(fullFieldContent)
				if match != nil {
					isField = true
					fieldNumber = parseInt(match[1])
				}

				if isField {
					// 是字段，添加到字段列表
					fields := currentContext["fields"].([]FieldInfo)
					fields = append(fields, FieldInfo{
						Number: fieldNumber,
						Lines:  currentFieldLines,
					})
					currentContext["fields"] = fields
					// 清空当前字段行
					currentFieldLines = []string{}
				} else if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "//") || strings.HasPrefix(strings.TrimSpace(line), "/*") {
					// 是注释或空行，保留在当前上下文中
					// 不做任何操作，继续保持在currentFieldLines中
					// 检查是否是多行注释的结束
					if strings.HasPrefix(strings.TrimSpace(line), "*/") {
						// 多行注释结束，将整个注释块添加到other_lines
						otherLines := currentContext["other_lines"].([]string)
						otherLines = append(otherLines, currentFieldLines...)
						currentContext["other_lines"] = otherLines
						currentFieldLines = []string{}
					}
				} else if strings.HasPrefix(strings.TrimSpace(line), "message ") && strings.HasSuffix(strings.TrimSpace(line), "{") {
					// 嵌套message定义，添加到other_lines
					otherLines := currentContext["other_lines"].([]string)
					otherLines = append(otherLines, currentFieldLines...)
					currentContext["other_lines"] = otherLines
					currentFieldLines = []string{}
				} else if isFieldLine(strings.TrimSpace(line)) {
					// 其他类型的字段行，添加到other_lines
					otherLines := currentContext["other_lines"].([]string)
					otherLines = append(otherLines, currentFieldLines...)
					currentContext["other_lines"] = otherLines
					currentFieldLines = []string{}
				} else if len(currentFieldLines) > 0 {
					// 其他情况，保留在currentFieldLines中直到找到分号或空行
					// 检查是否应该将累积的行添加到other_lines
					shouldAddToOther := true
					for _, fieldLine := range currentFieldLines {
						trimmed := strings.TrimSpace(fieldLine)
						if trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") && !strings.HasPrefix(trimmed, "*") {
							// 如果不是注释行，检查是否包含字段定义
							if fieldNumberRegex.MatchString(trimmed) {
								shouldAddToOther = false
								break
							}
						}
					}

					if shouldAddToOther {
						otherLines := currentContext["other_lines"].([]string)
						otherLines = append(otherLines, currentFieldLines...)
						currentContext["other_lines"] = otherLines
						currentFieldLines = []string{}
					}
				}
			}
		} else {
			// 4. Lines not inside any message
			finalOutput = append(finalOutput, line)
		}
	}

	// 处理文件末尾可能剩余的行
	if len(currentFieldLines) > 0 {
		finalOutput = append(finalOutput, currentFieldLines...)
	}

	return strings.Join(finalOutput, "\n")
}

// isContinuousSequence checks if field numbers form a continuous sequence starting from 1
func isContinuousSequence(fields []FieldInfo) bool {
	for i, field := range fields {
		if field.Number != i+1 {
			return false
		}
	}
	return true
}

// hasDuplicateFieldNumbers checks if there are duplicate field numbers in a message
func hasDuplicateFieldNumbers(fields []FieldInfo) bool {
	seen := make(map[int]bool)
	for _, field := range fields {
		if seen[field.Number] {
			return true
		}
		seen[field.Number] = true
	}
	return false
}

// getNextAvailableFieldNumber finds the next available field number
func getNextAvailableFieldNumber(fields []FieldInfo) int {
	maxNumber := 0
	for _, field := range fields {
		if field.Number > maxNumber {
			maxNumber = field.Number
		}
	}
	return maxNumber + 1
}

func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// isFieldLine checks if a line contains a field definition
func isFieldLine(line string) bool {
	// Skip empty lines, comments, and other non-field lines
	trimmed := strings.TrimSpace(line)
	return trimmed != "" &&
		!strings.HasPrefix(trimmed, "//") &&
		!strings.HasPrefix(trimmed, "/*") &&
		!strings.HasPrefix(trimmed, "*") &&
		!strings.HasPrefix(trimmed, "option ") &&
		strings.Contains(trimmed, "=") &&
		strings.Contains(trimmed, ";")
}
