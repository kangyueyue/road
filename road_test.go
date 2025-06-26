package road

import (
	"testing"
)

// TestRoad
func TestRoad(t *testing.T) {
	cfg, err := InitRoad("./test.conf")
	if err != nil {
		t.Error(err)
	}
	t.Log(cfg)
}
