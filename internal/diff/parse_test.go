package diff

import "testing"

const simplePatch = `@@ -18,7 +18,7 @@ func (s *Service) Charge(ctx context.Context) error {
 	if inv == nil {
 		return ErrNoInvoice
 	}
-	total := inv.Subtotal + inv.Subtotal*taxRate
+	total := getSubtotal(inv) + applyTax(inv.Subtotal)
 	if total <= 0 {
 		return ErrEmptyInvoice
 	}`

func TestParseTracksLineNumbers(t *testing.T) {
	hunks, err := Parse(simplePatch)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(hunks) != 1 {
		t.Fatalf("got %d hunks, want 1", len(hunks))
	}

	h := hunks[0]
	if h.OldStart != 18 || h.OldCount != 7 || h.NewStart != 18 || h.NewCount != 7 {
		t.Errorf("header = -%d,%d +%d,%d, want -18,7 +18,7",
			h.OldStart, h.OldCount, h.NewStart, h.NewCount)
	}
	if h.Section != "func (s *Service) Charge(ctx context.Context) error {" {
		t.Errorf("Section = %q", h.Section)
	}

	var del, add Line
	for _, l := range h.Lines {
		switch l.Kind {
		case Delete:
			del = l
		case Add:
			add = l
		}
	}

	if del.OldLine != 21 {
		t.Errorf("deleted line OldLine = %d, want 21", del.OldLine)
	}
	if del.NewLine != 0 {
		t.Errorf("deleted line NewLine = %d, want 0", del.NewLine)
	}
	if del.Side() != SideLeft || del.AnchorLine() != 21 {
		t.Errorf("deleted anchor = %s:%d, want LEFT:21", del.Side(), del.AnchorLine())
	}

	if add.NewLine != 21 {
		t.Errorf("added line NewLine = %d, want 21", add.NewLine)
	}
	if add.Side() != SideRight || add.AnchorLine() != 21 {
		t.Errorf("added anchor = %s:%d, want RIGHT:21", add.Side(), add.AnchorLine())
	}
}

func TestParseContextAnchorsToRight(t *testing.T) {
	hunks, err := Parse(simplePatch)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	first := hunks[0].Lines[0]
	if first.Kind != Context {
		t.Fatalf("first line kind = %v, want Context", first.Kind)
	}
	if first.Side() != SideRight {
		t.Errorf("context Side = %s, want RIGHT", first.Side())
	}
	if first.OldLine != 18 || first.NewLine != 18 {
		t.Errorf("context lines = old %d new %d, want 18/18", first.OldLine, first.NewLine)
	}
}

func TestParseOmittedCounts(t *testing.T) {
	hunks, err := Parse("@@ -1 +1 @@\n-old\n+new")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(hunks) != 1 {
		t.Fatalf("got %d hunks, want 1", len(hunks))
	}
	if hunks[0].OldCount != 1 || hunks[0].NewCount != 1 {
		t.Errorf("counts = %d/%d, want 1/1", hunks[0].OldCount, hunks[0].NewCount)
	}
}

func TestParseIgnoresNoNewlineMarker(t *testing.T) {
	patch := "@@ -1,2 +1,2 @@\n a\n-b\n\\ No newline at end of file\n+c\n\\ No newline at end of file"

	hunks, err := Parse(patch)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	add, del := Stats(hunks)
	if add != 1 || del != 1 {
		t.Errorf("stats = +%d -%d, want +1 -1", add, del)
	}
	for _, l := range hunks[0].Lines {
		if l.Content == `\ No newline at end of file` {
			t.Error("no-newline marker was parsed as a content line")
		}
	}
}

func TestParseMultipleHunksContinueNumbering(t *testing.T) {
	patch := `@@ -1,2 +1,2 @@
 one
-two
+TWO
@@ -50,2 +50,3 @@
 fifty
+fifty-one`

	hunks, err := Parse(patch)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(hunks) != 2 {
		t.Fatalf("got %d hunks, want 2", len(hunks))
	}

	second := hunks[1]
	if second.Lines[0].NewLine != 50 {
		t.Errorf("second hunk first line = %d, want 50", second.Lines[0].NewLine)
	}
	if second.Lines[1].NewLine != 51 {
		t.Errorf("added line = %d, want 51", second.Lines[1].NewLine)
	}
}

func TestParseEmptyPatch(t *testing.T) {
	for _, patch := range []string{"", "   ", "\n"} {
		hunks, err := Parse(patch)
		if err != nil {
			t.Errorf("Parse(%q) returned error: %v", patch, err)
		}
		if len(hunks) != 0 {
			t.Errorf("Parse(%q) returned %d hunks, want 0", patch, len(hunks))
		}
	}
}

func TestParseBlankContextLine(t *testing.T) {
	patch := "@@ -1,3 +1,3 @@\n a\n\n-b\n+c"

	hunks, err := Parse(patch)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	blank := hunks[0].Lines[1]
	if blank.Kind != Context || blank.Content != "" {
		t.Errorf("blank line = kind %v content %q, want Context and empty", blank.Kind, blank.Content)
	}
	if blank.NewLine != 2 {
		t.Errorf("blank line NewLine = %d, want 2", blank.NewLine)
	}
}

func TestParseRejectsMalformedHeader(t *testing.T) {
	if _, err := Parse("@@ nonsense @@\n a"); err == nil {
		t.Error("expected an error for a malformed hunk header")
	}
}
