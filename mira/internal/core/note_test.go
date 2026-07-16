package core

import (
	"strings"
	"testing"
)

func ptr(s string) *string { return &s }

func TestCreateNoteInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateNoteInput
		wantErr bool
		wantMsg string // sous-chaîne attendue si wantErr
	}{
		{"valide", CreateNoteInput{Title: "Go", Content: "..."}, false, ""},
		{"titre vide", CreateNoteInput{Title: "  ", Content: "..."}, true, "title is required"},
		{"titre absent", CreateNoteInput{Content: "..."}, true, "title is required"},
		{"contenu vide", CreateNoteInput{Title: "Go", Content: "  "}, true, "content is required"},
		{"titre trop long", CreateNoteInput{Title: strings.Repeat("a", 201), Content: "..."}, true, "200 characters"},
		{"titre exactement 200", CreateNoteInput{Title: strings.Repeat("a", 200), Content: "..."}, false, ""},
		{"toutes erreurs cumulées", CreateNoteInput{}, true, "title is required; content is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("attendu une erreur, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("attendu nil, got %v", err)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("erreur = %q, attendu qu'elle contienne %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestPatchNoteInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   PatchNoteInput
		wantErr bool
	}{
		{"tout absent -> valide (aucun champ à modifier)", PatchNoteInput{}, false},
		{"titre renseigné non vide", PatchNoteInput{Title: ptr("Go")}, false},
		{"titre renseigné vide", PatchNoteInput{Title: ptr("  ")}, true},
		{"titre trop long", PatchNoteInput{Title: ptr(strings.Repeat("a", 201))}, true},
		{"contenu renseigné non vide", PatchNoteInput{Content: ptr("x")}, false},
		{"contenu renseigné vide", PatchNoteInput{Content: ptr("   ")}, true},
		{"tags seuls -> valide", PatchNoteInput{Tags: []string{"a", "b"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("attendu une erreur, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("attendu nil, got %v", err)
			}
		})
	}
}

func TestEnrichmentStatusConstants(t *testing.T) {
	// Ces valeurs sont figées par la contrainte CHECK de la migration SQL
	// (enrichment_status IN ('pending','done','failed')) : toute divergence
	// ici casserait les écritures en base.
	if EnrichmentPending != "pending" || EnrichmentDone != "done" || EnrichmentFailed != "failed" {
		t.Errorf("constantes = %q/%q/%q, attendu pending/done/failed",
			EnrichmentPending, EnrichmentDone, EnrichmentFailed)
	}
}
