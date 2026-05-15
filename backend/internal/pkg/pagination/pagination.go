package pagination

import "strconv"

type Query struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Search string `json:"search"`
	Sort   string `json:"sort"`
}

func New(pageRaw, limitRaw, search, sort string) Query {
	p, _ := strconv.Atoi(pageRaw)
	l, _ := strconv.Atoi(limitRaw)
	if p < 1 {
		p = 1
	}
	if l < 1 || l > 200 {
		l = 20
	}
	return Query{Page: p, Limit: l, Search: search, Sort: sort}
}
func (q Query) Offset() int { return (q.Page - 1) * q.Limit }
