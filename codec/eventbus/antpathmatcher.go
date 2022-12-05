package eventbus

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

const (
	_DEFAULT_PATH_SEPARATOR  string = "/"
	_CACHE_TURNOFF_THRESHOLD int    = 65536
	_VARIABLE_PATTERN        string = "\\{[^/]+?}"
)

var _WILDCARD_CHARS []byte = []byte{'*', '?', '{'}

type concurrentHashMap struct {
	sync.RWMutex
	m map[string]interface{}
}

func (m *concurrentHashMap) get(key string) interface{} {
	m.RLock()
	defer m.RUnlock()
	return m.m[key]
}

func (m *concurrentHashMap) put(key string, value interface{}) {
	m.Lock()
	defer m.Unlock()
	m.m[key] = value
}

func (m *concurrentHashMap) clear() {
	m.Lock()
	defer m.Unlock()
	m.m = map[string]interface{}{}
}

func (m *concurrentHashMap) size() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.m)
}

type AntPathMatcher struct {
	pathSeparator             string
	pathSeparatorPatternCache pathSeparatorPatternCache
	caseSensitive             bool
	trimTokens                bool
	cachePatterns             *bool
	tokenizedPatternCache     concurrentHashMap
	stringMatcherCache        concurrentHashMap
}

func NewAntPathMatcher() *AntPathMatcher {
	m := AntPathMatcher{
		pathSeparator:             _DEFAULT_PATH_SEPARATOR,
		pathSeparatorPatternCache: newPathSeparatorPatternCache(_DEFAULT_PATH_SEPARATOR),
		caseSensitive:             true,
		trimTokens:                false,
		tokenizedPatternCache:     concurrentHashMap{m: map[string]interface{}{}},
		stringMatcherCache:        concurrentHashMap{m: map[string]interface{}{}},
	}
	return &m
}
func (that *AntPathMatcher) deactivatePatternCache() {
	f := false
	that.cachePatterns = &f
	that.tokenizedPatternCache.clear()
	that.stringMatcherCache.clear()
}

func (that *AntPathMatcher) Match(pattern string, path string) bool {
	return that.doMatch(pattern, path, true, nil)
}

func (that *AntPathMatcher) MatchStart(pattern string, path string) bool {
	return that.doMatch(pattern, path, false, nil)
}

func (that *AntPathMatcher) doMatch(pattern string, path string, fullMatch bool, uriTemplateVariables *map[string]string) bool {
	if len(path) == 0 || strings.HasPrefix(path, that.pathSeparator) != strings.HasPrefix(pattern, that.pathSeparator) {
		return false
	}

	var pattDirs []string = that.tokenizePattern(pattern)
	if fullMatch && that.caseSensitive && !that.isPotentialMatch(path, pattDirs) {
		return false
	}

	var pathDirs []string = that.tokenizePath(path)
	var pattIdxStart int = 0
	var pattIdxEnd int = len(pattDirs) - 1
	var pathIdxStart int = 0
	var pathIdxEnd int = len(pathDirs) - 1

	// Match all elements up to the first **
	for pattIdxStart <= pattIdxEnd && pathIdxStart <= pathIdxEnd {
		pattDir := pattDirs[pattIdxStart]
		if pattDir == "**" {
			break
		}
		if !that.matchStrings(pattDir, pathDirs[pathIdxStart], uriTemplateVariables) {
			return false
		}
		pattIdxStart++
		pathIdxStart++
	}

	if pathIdxStart > pathIdxEnd {
		// Path is exhausted, only match if rest of pattern is * or **'s
		if pattIdxStart > pattIdxEnd {
			return strings.HasSuffix(pattern, that.pathSeparator) == strings.HasSuffix(path, that.pathSeparator)
		}
		if !fullMatch {
			return true
		}
		if pattIdxStart == pattIdxEnd && pattDirs[pattIdxStart] == ("*") && strings.HasSuffix(path, that.pathSeparator) {
			return true
		}
		for i := pattIdxStart; i <= pattIdxEnd; i++ {
			if pattDirs[i] != "**" {
				return false
			}
		}
		return true
	} else if pattIdxStart > pattIdxEnd {
		// String not exhausted, but pattern is. Failure.
		return false
	} else if !fullMatch && pattDirs[pattIdxStart] == "**" {
		// Path start definitely matches due to "**" part in pattern.
		return true
	}

	// up to last '**'
	for pattIdxStart <= pattIdxEnd && pathIdxStart <= pathIdxEnd {
		pattDir := pattDirs[pattIdxEnd]
		if pattDir == "**" {
			break
		}
		if !that.matchStrings(pattDir, pathDirs[pathIdxEnd], uriTemplateVariables) {
			return false
		}
		pattIdxEnd--
		pathIdxEnd--
	}
	if pathIdxStart > pathIdxEnd {
		// String is exhausted
		for i := pattIdxStart; i <= pattIdxEnd; i++ {
			if pattDirs[i] != "**" {
				return false
			}
		}
		return true
	}

	for pattIdxStart != pattIdxEnd && pathIdxStart <= pathIdxEnd {
		patIdxTmp := -1
		for i := pattIdxStart + 1; i <= pattIdxEnd; i++ {
			if pattDirs[i] == "**" {
				patIdxTmp = i
				break
			}
		}
		if patIdxTmp == pattIdxStart+1 {
			// '**/**' situation, so skip one
			pattIdxStart++
			continue
		}
		// Find the pattern between padIdxStart & padIdxTmp in str between
		// strIdxStart & strIdxEnd
		patLength := (patIdxTmp - pattIdxStart - 1)
		strLength := (pathIdxEnd - pathIdxStart + 1)
		foundIdx := -1

	strLoop:
		for i := 0; i <= strLength-patLength; i++ {
			for j := 0; j < patLength; j++ {
				subPat := pattDirs[pattIdxStart+j+1]
				subStr := pathDirs[pathIdxStart+i+j]
				if !that.matchStrings(subPat, subStr, uriTemplateVariables) {
					continue strLoop
				}
			}
			foundIdx = pathIdxStart + i
			break
		}

		if foundIdx == -1 {
			return false
		}

		pattIdxStart = patIdxTmp
		pathIdxStart = foundIdx + patLength
	}

	for i := pattIdxStart; i <= pattIdxEnd; i++ {
		if pattDirs[i] != ("**") {
			return false
		}
	}

	return true
}

func (that *AntPathMatcher) isPotentialMatch(path string, pattDirs []string) bool {
	if !that.trimTokens {
		pos := 0
		for _, pattDir := range pattDirs {
			skipped := that.skipSeparator(path, pos, that.pathSeparator)
			pos += skipped
			skipped = that.skipSegment(path, pos, pattDir)
			if skipped < len(pattDir) {
				return (skipped > 0 || (len(pattDir) > 0 && that.isWildcardChar(pattDir[0])))
			}
			pos += skipped
		}
	}
	return true
}

func (that *AntPathMatcher) skipSegment(path string, pos int, prefix string) int {
	skipped := 0
	for i := 0; i < len(prefix); i++ {
		c := prefix[i]
		if that.isWildcardChar(c) {
			return skipped
		}
		currPos := pos + skipped
		if currPos >= len(path) {
			return 0
		}
		if c == path[currPos] {
			skipped++
		}
	}
	return skipped
}

func (that *AntPathMatcher) skipSeparator(path string, pos int, separator string) int {
	skipped := 0
	for startsWith(path, separator, pos+skipped) {
		skipped += len(separator)
	}
	return skipped
}

func startsWith(value string, prefix string, toffset int) bool {
	ta := []byte(value)
	to := toffset
	pa := []byte(prefix)
	po := 0
	pc := len(pa)
	// Note: toffset might be near -1>>>1.
	if (toffset < 0) || (toffset > len(ta)-pc) {
		return false
	}
	pc = pc - 1
	for pc >= 0 {
		if ta[to] != pa[po] {
			return false
		}
		to = to + 1
		po = po + 1
		pc = pc - 1
	}
	return true
}

func (that *AntPathMatcher) isWildcardChar(c byte) bool {
	for _, candidate := range _WILDCARD_CHARS {
		if c == candidate {
			return true
		}
	}
	return false
}

func (that *AntPathMatcher) tokenizePattern(pattern string) []string {
	var tokenized *[]string
	cachePatterns := that.cachePatterns
	if cachePatterns == nil || *cachePatterns {
		cache := that.tokenizedPatternCache.get(pattern)
		if cache != nil {
			tok := cache.([]string)
			tokenized = &tok
		}
	}
	if tokenized == nil {
		t := that.tokenizePath(pattern)
		tokenized = &t
		if cachePatterns == nil && that.tokenizedPatternCache.size() >= _CACHE_TURNOFF_THRESHOLD {
			// Try to adapt to the runtime situation that we're encountering:
			// There are obviously too many different patterns coming in here...
			// So let's turn off the cache since the patterns are unlikely to be reoccurring.
			that.deactivatePatternCache()
			return *tokenized
		}
		if cachePatterns == nil || *cachePatterns {
			that.tokenizedPatternCache.put(pattern, *tokenized)
		}
	}
	return *tokenized
}

func (that *AntPathMatcher) tokenizePath(path string) []string {
	return tokenizeToStringArray(path, that.pathSeparator, that.trimTokens, true)
}

func tokenizeToStringArray(str string, delimiters string, trimTokens bool, ignoreEmptyTokens bool) []string {
	if len(str) == 0 {
		return []string{}
	}
	tokens := strings.Split(str, delimiters)
	var tokens1 []string
	for _, token := range tokens {
		if trimTokens {
			token = strings.TrimSpace(token)
		}
		if !ignoreEmptyTokens || len(token) > 0 {
			tokens1 = append(tokens1, token)
		}
	}
	return tokens1
}

func (that *AntPathMatcher) matchStrings(pattern string, str string,
	uriTemplateVariables *map[string]string) bool {
	m := that.getStringMatcher(pattern)
	return m.matchStrings(str, uriTemplateVariables)
}

func (that *AntPathMatcher) getStringMatcher(pattern string) antPathStringMatcher {
	var matcher *antPathStringMatcher
	cachePatterns := that.cachePatterns
	if cachePatterns == nil || *cachePatterns {
		cache := that.stringMatcherCache.get(pattern)
		if cache != nil {
			m := cache.(antPathStringMatcher)
			matcher = &m
		}
	}
	if matcher == nil {
		m := newAntPathStringMatcher(pattern, that.caseSensitive)
		matcher = &m
		if cachePatterns == nil && that.stringMatcherCache.size() >= _CACHE_TURNOFF_THRESHOLD {
			// Try to adapt to the runtime situation that we're encountering:
			// There are obviously too many different patterns coming in here...
			// So let's turn off the cache since the patterns are unlikely to be reoccurring.
			that.deactivatePatternCache()
			return *matcher
		}
		if cachePatterns == nil || *cachePatterns {
			that.stringMatcherCache.put(pattern, *matcher)
		}
	}
	return *matcher
}

func (that *AntPathMatcher) ExtractPathWithinPattern(pattern string, path string) string {
	patternParts := tokenizeToStringArray(pattern, that.pathSeparator, that.trimTokens, true)
	pathParts := tokenizeToStringArray(path, that.pathSeparator, that.trimTokens, true)
	var builder string
	pathStarted := false

	for segment := 0; segment < len(patternParts); segment++ {
		patternPart := patternParts[segment]
		if strings.Contains(patternPart, "*") || strings.Contains(patternPart, "?") {
			for ; segment < len(pathParts); segment++ {
				if pathStarted || (segment == 0 && !strings.HasPrefix(pattern, that.pathSeparator)) {
					builder += that.pathSeparator
				}
				builder += pathParts[segment]
				pathStarted = true
			}
		}
	}

	return builder
}

func (that *AntPathMatcher) ExtractUriTemplateVariables(pattern string, path string) (map[string]string, error) {
	var variables map[string]string = map[string]string{}
	result := that.doMatch(pattern, path, true, &variables)
	if !result {
		return nil, fmt.Errorf("pattern \"" + pattern + "\" is not a match for \"" + path + "\"")
	}
	return variables, nil
}

func (that *AntPathMatcher) Combine(pattern1 string, pattern2 string) (string, error) {
	if len(strings.TrimSpace(pattern1)) == 0 && len(strings.TrimSpace(pattern2)) == 0 {
		return "", nil
	}
	if len(strings.TrimSpace(pattern1)) == 0 {
		return pattern2, nil
	}
	if len(strings.TrimSpace(pattern2)) == 0 {
		return pattern1, nil
	}

	pattern1ContainsUriVar := strings.Contains(pattern1, "{")
	if pattern1 != pattern2 && !pattern1ContainsUriVar && that.Match(pattern1, pattern2) {
		// /* + /hotel -> /hotel ; "/*.*" + "/*.html" -> /*.html
		// However /user + /user -> /usr/user ; /{foo} + /bar -> /{foo}/bar
		return pattern2, nil
	}

	// /hotels/* + /booking -> /hotels/booking
	// /hotels/* + booking -> /hotels/booking
	if strings.HasSuffix(pattern1, that.pathSeparatorPatternCache.endsOnWildCard) {
		return that.concat(pattern1[0:len(pattern1)-2], pattern2), nil
	}

	// /hotels/** + /booking -> /hotels/**/booking
	// /hotels/** + booking -> /hotels/**/booking
	if strings.HasSuffix(pattern1, that.pathSeparatorPatternCache.endsOnDoubleWildCard) {
		return that.concat(pattern1, pattern2), nil
	}

	starDotPos1 := strings.Index(pattern1, "*.")
	if pattern1ContainsUriVar || starDotPos1 == -1 || that.pathSeparator == (".") {
		// simply concatenate the two patterns
		return that.concat(pattern1, pattern2), nil
	}

	ext1 := pattern1[starDotPos1+1:]
	dotPos2 := strings.Index(pattern2, ".")
	file2 := pattern2
	if dotPos2 != -1 {
		file2 = pattern2[0:dotPos2]
	}
	ext2 := ""
	if dotPos2 != -1 {
		ext2 = pattern2[:dotPos2]
	}
	ext1All := (ext1 == (".*") || len(ext1) == 0)
	ext2All := (ext2 == (".*") || len(ext2) == 0)
	if !ext1All && !ext2All {
		return "", fmt.Errorf("cannot combine patterns: %s vs %s", pattern1, pattern2)
	}
	ext := ext1
	if ext1All {
		ext = ext2
	}
	return file2 + ext, nil
}

func (that *AntPathMatcher) concat(path1 string, path2 string) string {

	path1EndsWithSeparator := strings.HasSuffix(path1, that.pathSeparator)
	path2StartsWithSeparator := strings.HasSuffix(path2, that.pathSeparator)

	if path1EndsWithSeparator && path2StartsWithSeparator {
		return path1 + path2[1:]
	} else if path1EndsWithSeparator || path2StartsWithSeparator {
		return path1 + path2
	} else {
		return path1 + that.pathSeparator + path2
	}
}

type antPathStringMatcher struct {
	pattern       *regexp.Regexp
	variableNames []string
}

const _GLOB_PATTERN string = "\\?|\\*|\\{((?:\\{[^/]+?}|[^/{}]|\\\\[{}])+?)}"
const _DEFAULT_VARIABLE_PATTERN string = "(.*)"

var _GLOB_PATTERN_REGEXP *regexp.Regexp

func init() {
	_GLOB_PATTERN_REGEXP = regexp.MustCompile(_GLOB_PATTERN)
}

func newAntPathStringMatcher(pattern string, caseSensitive bool) antPathStringMatcher {
	var that antPathStringMatcher
	var patternBuilder string
	re := _GLOB_PATTERN_REGEXP //regexp.Compile(_GLOB_PATTERN)
	groups := re.FindStringSubmatch(pattern)
	allIndex := re.FindAllIndex([]byte(pattern), -1)
	end := 0
	for index := range allIndex {
		patternBuilder += that.quote(pattern, end, allIndex[index][0])
		match := groups[0]
		if match == "?" {
			patternBuilder += "."
		} else if match == "*" {
			patternBuilder += ".*"
		} else if strings.HasPrefix(match, "{") && strings.HasSuffix(match, "}") {
			colonIdx := strings.Index(match, ":")
			if colonIdx == -1 {
				patternBuilder += _DEFAULT_VARIABLE_PATTERN
				that.variableNames = append(that.variableNames, groups[1])
			} else {
				variablePattern := match[colonIdx+1 : len(match)-1]
				patternBuilder += "("
				patternBuilder += string(variablePattern)
				patternBuilder += ")"
				variableName := match[1:colonIdx]
				that.variableNames = append(that.variableNames, variableName)
			}
		}
		end = allIndex[index][1]
	}
	patternBuilder += that.quote(pattern, end, len(pattern))
	if caseSensitive {
		that.pattern = regexp.MustCompile(patternBuilder)
	} else {
		that.pattern = regexp.MustCompile("(?i)" + patternBuilder)
	}
	return that
}

func (that *antPathStringMatcher) quote(s string, start int, end int) string {
	if start == end {
		return ""
	}
	return regexp.QuoteMeta(s[start:end])
}

func (that *antPathStringMatcher) matchStrings(str string, uriTemplateVariables *map[string]string) bool {
	re := that.pattern
	all := re.FindAll([]byte(str), -1)
	if len(all) > 0 {
		if uriTemplateVariables != nil {
			if len(that.variableNames) == len(all) {
				for i := 0; i < len(all); i++ {
					name := that.variableNames[i]
					value := all[i]
					(*uriTemplateVariables)[name] = string(value)
				}
			}
		}
		return true
	} else {
		return false
	}
}

type pathSeparatorPatternCache struct {
	endsOnWildCard       string
	endsOnDoubleWildCard string
}

func newPathSeparatorPatternCache(pathSeparator string) pathSeparatorPatternCache {
	p := pathSeparatorPatternCache{
		endsOnWildCard:       pathSeparator + "*",
		endsOnDoubleWildCard: pathSeparator + "*",
	}
	return p
}
