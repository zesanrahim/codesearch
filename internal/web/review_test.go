package web

import "testing"

func TestNormalizeRange(t *testing.T) {
	tests := []struct {
		name          string
		inStart       int
		inStartSide   string
		inLine        int
		inSide        string
		wantStart     int
		wantStartSide string
		wantLine      int
		wantSide      string
	}{
		{
			name:   "single line has no range",
			inLine: 42, inSide: "RIGHT",
			wantLine: 42, wantSide: "RIGHT",
		},
		{
			name:     "side defaults to RIGHT",
			inLine:   42,
			inSide:   "",
			wantLine: 42, wantSide: "RIGHT",
		},
		{
			name:    "valid range passes through",
			inStart: 127, inStartSide: "RIGHT", inLine: 144, inSide: "RIGHT",
			wantStart: 127, wantStartSide: "RIGHT", wantLine: 144, wantSide: "RIGHT",
		},
		{
			name:    "reversed range is swapped",
			inStart: 144, inStartSide: "RIGHT", inLine: 127, inSide: "RIGHT",
			wantStart: 127, wantStartSide: "RIGHT", wantLine: 144, wantSide: "RIGHT",
		},
		{
			name:    "startSide defaults to side",
			inStart: 10, inStartSide: "", inLine: 20, inSide: "LEFT",
			wantStart: 10, wantStartSide: "LEFT", wantLine: 20, wantSide: "LEFT",
		},
		{
			name:    "collapsed range on same side becomes single line",
			inStart: 42, inStartSide: "RIGHT", inLine: 42, inSide: "RIGHT",
			wantLine: 42, wantSide: "RIGHT",
		},
		{
			name:    "same line across sides stays a range",
			inStart: 42, inStartSide: "LEFT", inLine: 42, inSide: "RIGHT",
			wantStart: 42, wantStartSide: "LEFT", wantLine: 42, wantSide: "RIGHT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotStartSide, gotLine, gotSide := normalizeRange(
				tt.inStart, tt.inStartSide, tt.inLine, tt.inSide)

			if gotStart != tt.wantStart || gotStartSide != tt.wantStartSide ||
				gotLine != tt.wantLine || gotSide != tt.wantSide {
				t.Errorf("normalizeRange(%d,%q,%d,%q) = (%d,%q,%d,%q), want (%d,%q,%d,%q)",
					tt.inStart, tt.inStartSide, tt.inLine, tt.inSide,
					gotStart, gotStartSide, gotLine, gotSide,
					tt.wantStart, tt.wantStartSide, tt.wantLine, tt.wantSide)
			}
		})
	}
}

func TestNormalizeRangeNeverEmitsStartAfterLine(t *testing.T) {
	for _, pair := range [][2]int{{5, 1}, {100, 99}, {2, 1}} {
		start, _, line, _ := normalizeRange(pair[0], "RIGHT", pair[1], "RIGHT")
		if start > line {
			t.Errorf("normalizeRange(%d, %d) produced start %d after line %d",
				pair[0], pair[1], start, line)
		}
	}
}
