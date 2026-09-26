package domain

import "testing"

func TestFlatFolders(t *testing.T) {
	c := Collection{Folders: []Folder{
		{Name: "A", Folders: []Folder{{Name: "B", Folders: []Folder{{Name: "C"}}}}},
		{Name: "D"},
	}}
	flat := c.FlatFolders()
	want := []string{"A", "A / B", "A / B / C", "D"}
	depths := []int{0, 1, 2, 0}
	if len(flat) != len(want) {
		t.Fatalf("got %d folders", len(flat))
	}
	for i, w := range want {
		if flat[i].Path != w || flat[i].Depth != depths[i] {
			t.Fatalf("folder %d = %q depth %d", i, flat[i].Path, flat[i].Depth)
		}
	}
	flat[1].Remove()
	if len(c.Folders[0].Folders) != 0 || len(c.FlatFolders()) != 2 {
		t.Fatal("remove nested folder failed")
	}
}
