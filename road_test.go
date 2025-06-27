package road

import (
	"github.com/spf13/viper"
	"testing"
)

// TestRoad
func TestRoad(t *testing.T) {
	_, err := InitRoad("./test.conf")
	if err != nil {
		t.Error(err)
	}
	host := viper.Get("config.conf")
	t.Logf("conf:%+v", host)
}
