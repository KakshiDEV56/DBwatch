package collector

import "testing"

func TestCacheHitRatio(t *testing.T) {
	tests := []struct {
		name string
		s    CacheStats
		want float64
	}{
		{"no reads yet is 100", CacheStats{}, 100},
		{"all hits", CacheStats{BlocksHit: 90, BlocksRead: 0}, 100},
		{"mixed", CacheStats{BlocksHit: 75, BlocksRead: 25}, 75},
		{"all misses", CacheStats{BlocksHit: 0, BlocksRead: 10}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.HitRatio(); got != tt.want {
				t.Fatalf("HitRatio() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConnectionUtilization(t *testing.T) {
	if got := (ConnectionStats{}).UtilizationPercent(); got != 0 {
		t.Fatalf("zero max connections: got %v, want 0", got)
	}
	got := (ConnectionStats{Total: 25, MaxConnections: 100}).UtilizationPercent()
	if got != 25 {
		t.Fatalf("UtilizationPercent() = %v, want 25", got)
	}
}

func TestCapabilityHasExtension(t *testing.T) {
	s := CapabilityStats{Extensions: []string{"pg_stat_statements", "pgcrypto"}}
	if !s.HasExtension("pg_stat_statements") {
		t.Fatal("expected pg_stat_statements to be present")
	}
	if s.HasExtension("missing") {
		t.Fatal("did not expect missing extension")
	}
}
