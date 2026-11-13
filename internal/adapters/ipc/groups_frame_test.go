package ipc

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/foundation"
	"strings"
	"testing"
)

func TestGroupResponsesKeepFiveHundredUnicodeNamesWithinScopedFrame(t *testing.T) {
	j := newPPJourney(t)
	before := j.peer.Auth.Load()
	code := j.store.UpdateEquipment(context.Background(), j.client.profile, func(state *equipment.State) equipment.Code {
		for n := 0; n < 500; n++ {
			state.Groups = append(state.Groups, equipment.Group{ID: uuid.NewString(),
				Name: fmt.Sprintf("%03d", n) + strings.Repeat("界", 76)})
		}
		return equipment.OK
	})
	if code != equipment.OK {
		t.Fatal(code)
	}
	q := Request{Op: "inventory.groups", ExpectedProfile: j.client.profile,
		ExpectedActivation: j.client.activation, Equipment: &EquipmentInput{}}
	listed := j.client.Call(context.Background(), q)
	if listed.Code != foundation.OK || listed.Data == nil || len(listed.Data.Equipment.Groups) != 500 {
		t.Fatal("bounded group frame", listed.Code)
	}
	first := listed.Data.Equipment.Groups[0]
	q.Op = "inventory.group-rename"
	q.Equipment = &EquipmentInput{ID: first.ID, Operation: uuid.NewString(), Name: "Novo nome"}
	renamed := j.client.Call(context.Background(), q)
	if renamed.Code != foundation.OK || renamed.Data == nil || len(renamed.Data.Equipment.Groups) != 500 {
		t.Fatal("rename group frame", renamed.Code)
	}
	if renamed.Data.Equipment.Groups[0].ID != first.ID || renamed.Data.Equipment.Groups[0].Name != "Novo nome" {
		t.Fatal("rename changed identity")
	}
	if j.peer.Auth.Load() != before {
		t.Fatal("group operation contacted device")
	}
}
