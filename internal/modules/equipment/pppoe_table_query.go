package equipment

import "unicode/utf8"

func validPPTableQuery(q *PPQuery) bool {
	if q.PageSize != 0 && q.PageSize != 25 && q.PageSize != 50 && q.PageSize != 100 && q.PageSize != 200 {
		return false
	}
	for _, v := range []string{q.Plan, q.VLAN, q.Service} {
		if !utf8.ValidString(v) || utf8.RuneCountInString(v) > 200 {
			return false
		}
	}
	return q.Duplicity == "" || q.Duplicity == "with" || q.Duplicity == "without"
}
