package handler

import "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"

// buildPagination converts limit/offset style pagination into the shared response pagination metadata.
func buildPagination(total int64, limit, offset int) response.Pagination {
	if limit <= 0 {
		limit = 1
	}
	if offset < 0 {
		offset = 0
	}
	page := offset/limit + 1
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}
	return response.Pagination{
		Page:       page,
		PageSize:   limit,
		Total:      int(total),
		TotalPages: totalPages,
	}
}

type statusMessageResponse struct {
	Message string `json:"message"`
}
