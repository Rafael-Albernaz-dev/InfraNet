package storage

import (
	"encoding/json"
	"infradesktop/internal/modules/equipment"
)

// Automatic collections avoid copying indexed columns and the observation's
// shared sample into every JSON row. The view restores the public contract.
func marshalStoredPPRow(o *equipment.PPObservation, row equipment.PPRow) ([]byte, error) {
	body, err := json.Marshal(row)
	if err != nil || !o.Provenance.Automatic {
		return body, err
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(body, &fields); err != nil {
		return nil, err
	}
	for _, key := range []string{"row_id", "login", "address", "caller_id"} {
		delete(fields, key)
	}
	if row.State == "ONLINE" {
		delete(fields, "state")
	}
	if row.Duplicates <= 1 {
		delete(fields, "duplicates")
	}
	if row.Sample == o.Provenance.Sample && row.Sample.Known {
		delete(fields, "sample")
		fields["sample_shared"] = json.RawMessage("true")
	}
	return json.Marshal(fields)
}
