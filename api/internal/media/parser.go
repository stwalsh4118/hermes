package media

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ParseResult contains extracted metadata from a filename
type ParseResult struct {
	ShowName    *string // Extracted show name (nil if not found)
	Season      *int    // Season number (nil if not found)
	Episode     *int    // Episode number (nil if not found)
	Title       string  // Generated display title
	RawFilename string  // Original filename for reference
}

// Patterns for matching show/season/episode in filenames
var (
	// Pattern 1: "Show Name - S01E01" or "Show Name - S01E01 - Episode Title"
	patternDashFormat = regexp.MustCompile(`(?i)^(.+?)\s*-\s*[Ss](\d+)[Ee](\d+)`)

	// Pattern 2: "Show.Name.S01E01" or "Show Name S01E01" or "Show_Name_S01E01"
	patternStandardFormat = regexp.MustCompile(`(?i)^(.+?)[._ ][Ss](\d+)[Ee](\d+)`)

	// Pattern 3: "Show.Name.1x01" (alternate format)
	patternAlternateFormat = regexp.MustCompile(`(?i)^(.+?)[._ ](\d+)x(\d+)`)

	// Pattern for extracting season from directory name: "Season 1", "Season 01", "S01"
	patternSeasonDir = regexp.MustCompile(`(?i)(?:season|s)[\s.]?(\d+)`)

	// Pattern for extracting episode number from filename: "01 -", "Episode 01", "E01"
	patternEpisodeFile = regexp.MustCompile(`(?i)^(\d+)\s*-|^[Ee](\d+)|^episode[\s.]?(\d+)`)
)

// ParseFilename extracts show name, season, and episode from a filename or path
// It tries multiple common patterns and returns the first successful match
func ParseFilename(fullPath string) ParseResult {
	result := ParseResult{
		RawFilename: fullPath,
	}

	// Get the directory and filename components
	dir := filepath.Dir(fullPath)
	filename := filepath.Base(fullPath)

	// Remove extension
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	// Try standard patterns on the filename
	if tryStandardPatterns(nameWithoutExt, &result) {
		result.Title = generateTitle(&result)
		return result
	}

	// Try directory-based parsing
	if tryDirectoryPatterns(dir, nameWithoutExt, &result) {
		result.Title = generateTitle(&result)
		return result
	}

	// No pattern matched - use cleaned filename as title
	result.Title = cleanShowName(nameWithoutExt)
	return result
}

// tryStandardPatterns tries common filename patterns
func tryStandardPatterns(filename string, result *ParseResult) bool {
	// Try Pattern 1: "Show Name - S01E01"
	if matches := patternDashFormat.FindStringSubmatch(filename); matches != nil {
		showName := cleanShowName(matches[1])
		season := parseInt(matches[2])
		episode := parseInt(matches[3])

		result.ShowName = &showName
		result.Season = season
		result.Episode = episode
		return true
	}

	// Try Pattern 2: "Show.Name.S01E01"
	if matches := patternStandardFormat.FindStringSubmatch(filename); matches != nil {
		showName := cleanShowName(matches[1])
		season := parseInt(matches[2])
		episode := parseInt(matches[3])

		result.ShowName = &showName
		result.Season = season
		result.Episode = episode
		return true
	}

	// Try Pattern 3: "Show.Name.1x01"
	if matches := patternAlternateFormat.FindStringSubmatch(filename); matches != nil {
		showName := cleanShowName(matches[1])
		season := parseInt(matches[2])
		episode := parseInt(matches[3])

		result.ShowName = &showName
		result.Season = season
		result.Episode = episode
		return true
	}

	return false
}

// tryDirectoryPatterns tries to extract info from directory structure
// Example: "Show Name/Season 1/01 - Episode.mp4"
func tryDirectoryPatterns(dirPath string, filename string, result *ParseResult) bool {
	if dirPath == "." || dirPath == "/" || dirPath == "" {
		return false
	}

	// Split directory path
	parts := strings.Split(filepath.ToSlash(dirPath), "/")
	if len(parts) < 2 {
		return false
	}

	// Try to extract season from second-to-last directory
	seasonDir := parts[len(parts)-1]
	if matches := patternSeasonDir.FindStringSubmatch(seasonDir); matches != nil {
		season := parseInt(matches[1])
		result.Season = season

		// Show name is the parent directory
		if len(parts) >= 2 {
			showName := cleanShowName(parts[len(parts)-2])
			result.ShowName = &showName
		}

		// Try to extract episode number from filename
		if matches := patternEpisodeFile.FindStringSubmatch(filename); matches != nil {
			// Episode number could be in any of the capture groups
			for i := 1; i < len(matches); i++ {
				if matches[i] != "" {
					episode := parseInt(matches[i])
					result.Episode = episode
					break
				}
			}
			return true
		}
	}

	return false
}

// cleanShowName normalizes show names by replacing separators with spaces
func cleanShowName(name string) string {
	// Replace dots and underscores with spaces
	cleaned := strings.ReplaceAll(name, ".", " ")
	cleaned = strings.ReplaceAll(cleaned, "_", " ")

	// Trim whitespace
	cleaned = strings.TrimSpace(cleaned)

	// Collapse multiple spaces
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")

	return cleaned
}

// parseInt safely converts a string to an int pointer
func parseInt(s string) *int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &val
}

// generateTitle creates a display title from parsed information
func generateTitle(result *ParseResult) string {
	// If we have show name and episode info, format as "Show Name - S01E01"
	if result.ShowName != nil && result.Season != nil && result.Episode != nil {
		return formatEpisodeTitle(*result.ShowName, *result.Season, *result.Episode)
	}

	// If we only have show name, use that
	if result.ShowName != nil {
		return *result.ShowName
	}

	// Otherwise use raw filename without extension
	name := filepath.Base(result.RawFilename)
	ext := filepath.Ext(name)
	return cleanShowName(strings.TrimSuffix(name, ext))
}

// formatEpisodeTitle creates a formatted title string
func formatEpisodeTitle(showName string, season int, episode int) string {
	return strings.TrimSpace(showName) + " - " + formatSeasonEpisode(season, episode)
}

// formatSeasonEpisode formats season and episode as S01E01
func formatSeasonEpisode(season int, episode int) string {
	return "S" + padNumber(season) + "E" + padNumber(episode)
}

// padNumber pads a number to 2 digits
func padNumber(num int) string {
	if num < 10 {
		return "0" + strconv.Itoa(num)
	}
	return strconv.Itoa(num)
}
