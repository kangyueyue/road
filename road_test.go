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
	host := cfg.viper.Get("mysql.host")
	t.Logf("mysql host:%s",host)
}
