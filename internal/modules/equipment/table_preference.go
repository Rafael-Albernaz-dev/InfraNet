package equipment

// TablePreference is the local arrangement shared by inventory and the UI tables.
type TablePreference struct {
	Columns  []string          `json:"columns"`
	Filters  map[string]string `json:"filters"`
	Search   string            `json:"search"`
	Sort     *TableSort        `json:"sort"`
	Manual   []string          `json:"manual"`
	PageSize int               `json:"page_size"`
}
type TableSort struct {
	Column string `json:"column"`
	Desc   bool   `json:"desc"`
}
