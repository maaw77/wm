// Package dto содержит структуры запросов и ответов HTTP API сервиса.
package dto

// CheckLinksRequest описывает входной JSON для эндпоинта POST /links вида:
// { "links": ["google.com", "example.com"] }.
type CheckLinksRequest struct {
	Links []string `json:"links"`
}

// CheckLinksResponse описывает ответ эндпоинта POST /links вида:
// { "links": { "url": "status" }, "links_num": N }.
type CheckLinksResponse struct {
	Links    map[string]string `json:"links"`
	LinksNum uint64            `json:"links_num"`
}

// ReportRequest описывает входной JSON для эндпоинта POST /report вида:
// { "links_list": [1, 2, 3] }.
type ReportRequest struct {
	LinksList []uint64 `json:"links_list"`
}
