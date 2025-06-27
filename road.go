package road

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Road nacos配置初始化
type Road struct {
	viper       *viper.Viper                // viper配置
	nacosClient config_client.IConfigClient // nacos客户端
	cfg         *Config                     // nacos配置
}

// Config nacos配置
type Config struct {
	BaseConfig  *BaseConfig  `toml:"base_config"`  // 基础配置
	NacosServer *NacosServer `toml:"nacos_server"` // nacos服务端配置
	NacosClient *NacosClient `toml:"nacos_client"` // nacos客户端配置
}

// BaseConfig 基础配置
type BaseConfig struct {
	CacheDir string `toml:cache_dir`
	DataId   string `toml:"data_id"`
	Group    string `toml:"group"`
}

// NacosServer nacos服务端配置
type NacosServer struct {
	IpAddr string `toml:"ip_addr"`
	Port   uint64 `toml:"port"`
	Scheme string `toml:"scheme"`
}

// NacosClient nacos客户端配置
type NacosClient struct {
	NamespaceId         string `toml:"namespace_id"`
	TimeoutMs           uint64 `toml:"timeout_ms"`
	NotLoadCacheAtStart bool   `toml:"not_load_cache_at_start"`
	LogDir              string `toml:"log_dir"`
	CacheDir            string `toml:"cache_dir"`
	LogLevel            string `toml:"log_level"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	DefaultBaseConfig := &BaseConfig{
		CacheDir: "tmp/nacos/config",
	}
	DefaultNacosServer := &NacosServer{
		Scheme: "http",
	}
	DefaultNacosClient := &NacosClient{
		TimeoutMs:           5000,
		NotLoadCacheAtStart: false,
		LogDir:              "tmp/nacos/log",
		CacheDir:            "tmp/nacos/cache",
		LogLevel:            "debug",
	}
	DefaultConfig := &Config{
		BaseConfig:  DefaultBaseConfig,
		NacosServer: DefaultNacosServer,
		NacosClient: DefaultNacosClient,
	}
	return DefaultConfig
}

// NewConfigByFile 通过文件初始化config
func NewConfigByFile(f string) (c *Config, err error) {
	cf := DefaultConfig()
	if _, err := toml.DecodeFile(f, cf); err != nil {
		return nil, err
	}
	return cf, nil
}

// NewNacosClient 初始化nacos客户端
func NewNacosClient(cfg Config) (config_client.IConfigClient, error) {
	sc := []constant.ServerConfig{{
		IpAddr: cfg.NacosServer.IpAddr,
		Port:   cfg.NacosServer.Port,
		Scheme: cfg.NacosServer.Scheme,
	}}

	// 构建ClientConfig
	cc := constant.ClientConfig{
		NamespaceId:         cfg.NacosClient.NamespaceId,
		TimeoutMs:           cfg.NacosClient.TimeoutMs,
		NotLoadCacheAtStart: cfg.NacosClient.NotLoadCacheAtStart,
		LogDir:              cfg.NacosClient.LogDir,
		CacheDir:            cfg.NacosClient.CacheDir,
		LogLevel:            cfg.NacosClient.LogLevel,
	}

	param := vo.NacosClientParam{
		ClientConfig:  &cc,
		ServerConfigs: sc,
	}
	return clients.NewConfigClient(param)
}

// NewcRoad 初始化配置
func NewcRoad(viper *viper.Viper, cfg *Config) (*Road, error) {
	client, err := NewNacosClient(*cfg)
	if err != nil {
		return nil, err
	}
	r := &Road{
		viper:       viper,
		nacosClient: client,
		cfg:         cfg,
	}
	// 启动监控
	r.watch()
	return r, nil
}

// watch 监听同步
func (r *Road) watch() {
	var (
		err     error
		content string
	)
	// 从nacos读取配置
	content, err = r.nacosClient.GetConfig(vo.ConfigParam{
		DataId: r.cfg.BaseConfig.DataId,
		Group:  r.cfg.BaseConfig.Group,
	})
	if err != nil {
		logrus.Fatalf("读取配置文件失败: %v", err)
		panic(err)
	}
	// 写入到本地，缓存配置
	r.createConfigCache(content)

	r.viper.SetConfigType("toml")
	if err = r.viper.ReadConfig(bytes.NewBuffer([]byte(content))); err != nil {
		logrus.Fatalf("读取配置文件失败: %v", err)
		panic(err)
	}
	// 监听
	err = r.nacosClient.ListenConfig(vo.ConfigParam{
		DataId: r.cfg.BaseConfig.DataId,
		Group:  r.cfg.BaseConfig.Group,
		OnChange: func(namespace, group, dataId, data string) {
			logrus.Info("检测到配置变更，重新加载配置")
			// 重新加载配置
			if err = r.viper.ReadConfig(bytes.NewBuffer([]byte(data))); err != nil {
				logrus.Errorf("重新加载配置失败: %v", err)
				return
			}
			// 写入到本地，缓存配置
			r.createConfigCache(content) // 可以在这里添加配置变更后的处理逻辑
			logrus.Info("配置重新加载成功")
		},
	})

	if err != nil {
		logrus.Warnf("配置监听设置失败: %v", err)
	}
}

// createConfigCache 创建配置缓存
func (r *Road) createConfigCache(content string) {
	cacheDir := r.cfg.BaseConfig.CacheDir
	// 判断是否存在，不存在创建
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		if err = os.MkdirAll(cacheDir, 0755); err != nil {
			return
		}
	}
	cacheFile := filepath.Join(cacheDir, r.cfg.BaseConfig.DataId)
	if err := os.WriteFile(cacheFile, []byte(content), 0644); err != nil {
		return
	}
}
