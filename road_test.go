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
	v1 := viper.Get("config.conf")
	v2 := viper.Get("config.yaml")
	t.Logf("conf:%+v", v1)
	t.Logf("conf:%+v", v2)
}
