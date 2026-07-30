package ws

import "testing"

func TestActiveVisitorCountsByKefu(t *testing.T) {
	visitors := map[string]*User{
		"visitor-1": {Id: "visitor-1", Ent_id: "1", To_id: "WGG1"},
		"visitor-2": {Id: "visitor-2", Ent_id: "1", To_id: "WGG1"},
		"visitor-3": {Id: "visitor-3", Ent_id: "1", To_id: "WGG2"},
		"visitor-4": {Id: "visitor-4", Ent_id: "2", To_id: "OTHER"},
	}

	counts := activeVisitorCountsByKefu("1", visitors)
	if counts["WGG1"] != 2 || counts["WGG2"] != 1 {
		t.Fatalf("ActiveVisitorCountsByKefu() = %#v, want WGG1=2 and WGG2=1", counts)
	}
	if _, ok := counts["OTHER"]; ok {
		t.Fatalf("other enterprise leaked into counts: %#v", counts)
	}
}
