package road

import (
	"fmt"

	"github.com/spf13/viper"
)

// InitRoad 初始化
func InitRoad(f string) (*Road, error) {
	r, err := NewConfigByFile(f)
	if err != nil {
		return nil, fmt.Errorf("NewConfig:%w", err)
	}
	cfg, err := NewcRoad(viper.GetViper(), r)
	if err != nil {
		return nil, fmt.Errorf("NewRoad err:%+v", err)
	}
	return cfg, nil
}
