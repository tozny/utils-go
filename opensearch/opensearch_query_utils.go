package opensearch

import "time"

type BoolQueryClause struct {
	Query BoolQuery `json:"query"`
}

type BoolQuery struct {
	Bool Bool `json:"bool"`
}

type BoolClause struct {
	Bool Bool `json:"bool"`
}

type Bool struct {
	Must    []interface{} `json:"must,omitempty"`
	MustNot []interface{} `json:"must_not,omitempty"`
	Should  []interface{} `json:"should,omitempty"`
	Filter  []interface{} `json:"filter,omitempty"`
}

type MatchQueryClause struct {
	Query MatchQuery `json:"query"`
}

type MatchQuery struct {
	Match map[string]interface{} `json:"match"`
}

type NestedClause struct {
	Nested Nested `json:"nested"`
}

type Nested struct {
	Path  string    `json:"path"`
	Query BoolQuery `json:"query"`
}

func NewBoolQueryClause(queryInput BoolQuery) BoolQueryClause {
	return BoolQueryClause{Query: queryInput}
}

func NewBoolClause(boolInput Bool) BoolClause {
	return BoolClause{Bool: boolInput}
}

func NewMatchQueryClause(field string, value string) MatchQueryClause {
	return MatchQueryClause{Query: MatchQuery{Match: BuildMatchClause(field, value)}}
}

func NewNestedClause(nested Nested) NestedClause {
	return NestedClause{Nested: nested}
}

func BuildExistsClause(existsValue string) map[string]interface{} {
	return map[string]interface{}{
		"exists": map[string]interface{}{
			"field": existsValue,
		},
	}
}

func BuildFuzzyClause(field string, value string) map[string]interface{} {
	return map[string]interface{}{
		"fuzzy": map[string]interface{}{
			field: map[string]interface{}{
				"value": value,
			},
		},
	}
}

func BuildWildcardClause(field string, value string) map[string]interface{} {
	return map[string]interface{}{
		"wildcard": map[string]interface{}{
			field: map[string]interface{}{
				"value": value,
			},
		},
	}
}

func BuildRegexpClause(field string, value string) map[string]interface{} {
	return map[string]interface{}{
		"regexp": map[string]interface{}{
			field: map[string]interface{}{
				"value": value,
			},
		},
	}
}

func BuildMatchClause(field string, value string) map[string]interface{} {
	return map[string]interface{}{
		"match": map[string]interface{}{
			field: value,
		},
	}
}

func BuildTermClause(field string, value interface{}) map[string]interface{} {
	return map[string]interface{}{
		"term": map[string]interface{}{
			field: value,
		},
	}
}

func BuildRangeClause(field string, gte, lte time.Time) map[string]interface{} {
	rangeQuery := map[string]interface{}{}

	if !gte.IsZero() {
		rangeQuery["gte"] = gte.Format(time.RFC3339Nano)
	}
	if !lte.IsZero() {
		rangeQuery["lte"] = lte.Format(time.RFC3339Nano)
	}

	return map[string]interface{}{
		"range": map[string]interface{}{
			field: rangeQuery,
		},
	}
}
