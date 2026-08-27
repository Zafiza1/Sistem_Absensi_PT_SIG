// Package pagination provides the single page/page_size query-parameter
// convention and response envelope used by every list endpoint
// (departments, positions, shifts, employees, schedules, devices, ...).
package pagination

import (
	"net/url"
	"strconv"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Params is a parsed, validated page/page_size pair.
type Params struct {
	Page     int
	PageSize int
}

// Offset returns the SQL OFFSET for these params; combine with PageSize as
// the SQL LIMIT.
func (p Params) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// FromQuery parses "page" and "page_size" from request query parameters,
// applying defaults for missing/invalid values and clamping page_size to
// MaxPageSize so a client can't force an unbounded table scan.
func FromQuery(q url.Values) Params {
	page, err := strconv.Atoi(q.Get("page"))
	if err != nil || page < 1 {
		page = DefaultPage
	}

	pageSize, err := strconv.Atoi(q.Get("page_size"))
	if err != nil || pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	return Params{Page: page, PageSize: pageSize}
}

// Meta is embedded in every paginated list response.
type Meta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// NewMeta builds a Meta from the request params and the total row count
// (typically from a `COUNT(*) OVER()` window function alongside the page
// query, avoiding a second round-trip).
func NewMeta(p Params, totalItems int64) Meta {
	totalPages := int((totalItems + int64(p.PageSize) - 1) / int64(p.PageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	return Meta{
		Page:       p.Page,
		PageSize:   p.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}
