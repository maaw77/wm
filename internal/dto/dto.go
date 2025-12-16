package dto

// checkLinksRequest описывает входной JSON { "links": [...] }.
type CheckLinksRequest struct {
	Links []string `json:"links"`
}

// checkLinksResponse описывает ответ:
// { "links": { "url": "status", ... }, "links_num": N }.
type CheckLinksResponse struct {
	Links    map[string]string `json:"links"`
	LinksNum uint64            `json:"links_num"`
}

// reportRequest описывает входной JSON { "links_list": [1,2,...] }.
type ReportRequest struct {
	LinksList []uint64 `json:"links_list"`
}
