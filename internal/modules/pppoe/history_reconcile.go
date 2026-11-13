package pppoe

import "infradesktop/internal/modules/equipment"

type OpenSession struct {
	ID         int64
	Key, Login string
}

type HistoryPlan struct {
	Connect    []PPRow
	Disconnect []int64
	Rebind     map[int64]string
}

// During a blind interval a surviving unique login keeps its open connection.
// Neither a changed remote handle nor a daemon restart proves a disconnection.
func ReconcileHistory(open []OpenSession, rows []equipment.PPRow, gap bool) HistoryPlan {
	plan := HistoryPlan{Rebind: map[int64]string{}}
	byKey := map[string]OpenSession{}
	byLogin := map[string][]OpenSession{}
	currentLogins := map[string]int{}
	for _, before := range open {
		byKey[before.Key] = before
		byLogin[before.Login] = append(byLogin[before.Login], before)
	}
	for _, row := range rows {
		if row.State != "OFFLINE" {
			currentLogins[row.Login]++
		}
	}
	kept := map[int64]bool{}
	for _, row := range rows {
		if row.State == "OFFLINE" {
			continue
		}
		key := SessionKey(row)
		if before, ok := byKey[key]; ok {
			kept[before.ID] = true
			continue
		}
		previous := byLogin[row.Login]
		if gap && row.Login != "" && currentLogins[row.Login] == 1 && len(previous) == 1 {
			kept[previous[0].ID] = true
			plan.Rebind[previous[0].ID] = key
			continue
		}
		plan.Connect = append(plan.Connect, row)
	}
	for _, before := range open {
		if !kept[before.ID] {
			plan.Disconnect = append(plan.Disconnect, before.ID)
		}
	}
	return plan
}
