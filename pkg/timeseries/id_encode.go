package timeseries

import "strings"

// esIndexPart ES 索引名片段：去空白后转小写（ES 索引名要求小写）。
// 与业务 ID 大小写折叠：MyProduct / MYPRODUCT / myproduct 共用同一索引段。
// 允许 '-'；不做大小写转义，避免索引名膨胀。
func esIndexPart(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// tdNamePart TDengine 表名片段：小写，并将 '-' 换成 '_'（TD 标识符更稳妥）。
// 例：MyProduct → myproduct；prod-a → prod_a。
func tdNamePart(s string) string {
	return strings.ReplaceAll(esIndexPart(s), "-", "_")
}
