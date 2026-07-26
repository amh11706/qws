package qws

import (
	"encoding/json"
	"testing"
)

func TestUserRejectsServerOwnedFields(t *testing.T) {
	body := `{"Id":1,"AdminLvl":4,"Inventory":7,"Locked":false,"email":"a@b.c","password":"x"}`
	u := &User{}
	if err := json.Unmarshal([]byte(body), u); err != nil {
		t.Fatal(err)
	}
	if u.Id != 0 || u.AdminLvl != 0 || u.Inventory != 0 || u.Locked {
		t.Fatalf("client set a server owned field: id=%d admin=%d inv=%d locked=%v",
			u.Id, u.AdminLvl, u.Inventory, u.Locked)
	}
	if u.Email != "a@b.c" || u.Pass != "x" {
		t.Fatalf("client fields did not decode: %q %q", u.Email, u.Pass)
	}
}
