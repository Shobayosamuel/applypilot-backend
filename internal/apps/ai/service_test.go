package ai

import "testing"

func TestCleanFieldValues(t *testing.T) {
	fields := []FormField{
		{ID: "a", Type: "select", Options: []string{"Yes", "No"}},
		{ID: "b", Type: "text"},
		{ID: "c", Type: "radio", Options: []string{"0-2", "3-5"}},
	}
	got := cleanFieldValues(fields, []FieldValue{
		{ID: "a", Value: "yes"},   // normalised to option spelling
		{ID: "b", Value: "Lagos"}, // free text kept
		{ID: "c", Value: "Seven"}, // not an option: dropped
		{ID: "zzz", Value: "x"},   // unknown field: dropped
		{ID: "b", Value: ""},      // empty: dropped
	})
	if len(got) != 2 || got[0].Value != "Yes" || got[1].Value != "Lagos" {
		t.Fatalf("got %+v", got)
	}
}
