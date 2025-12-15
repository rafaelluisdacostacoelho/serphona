// Package pagination provides helpers for pagination parameters and responses.
package pagination

// Params represents common pagination query parameters.
type Params struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// Normalize ensures sane defaults and limits to avoid abusive page sizes.
func (p Params) Normalize() Params {
	page := p.Page
	if page < 1 {
		page = 1
	}
	pageSize := p.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return Params{
		Page:     page,
		PageSize: pageSize,
	}
}

// Offset returns the offset for SQL queries based on normalized pagination params.
func (p Params) Offset() int {
	n := p.Normalize()
	return (n.Page - 1) * n.PageSize
}

// Meta represents pagination metadata in responses.
type Meta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// ComputeMeta calculates pagination metadata given total items and params.
func ComputeMeta(total int64, params Params) Meta {
	n := params.Normalize()
	totalPages := int(total) / n.PageSize
	if int(total)%n.PageSize != 0 {
		totalPages++
	}
	return Meta{
		Page:       n.Page,
		PageSize:   n.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}
