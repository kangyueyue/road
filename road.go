package road

import (
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
	CacheDir string `toml:"cache_dir"`
	PageSize int    `toml:"page_size"`
	Group    string `toml:"group"`
	Search   string `toml:"search"`
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
		Search:   "accurate",
		PageSize: 10,
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

// NewRoad 初始化配置
func NewRoad(viper *viper.Viper, cfg *Config) (*Road, error) {
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
	// 从nacos得到所有dataId
	dataIds := r.getDataIds()
	for _, dataId := range dataIds {
		content, err = r.nacosClient.GetConfig(vo.ConfigParam{
			DataId: dataId,
			Group:  r.cfg.BaseConfig.Group,
		})
		if err != nil {
			logrus.Fatalf("读取配置文件失败: %v", err)
			panic(err)
		}
		// 写入到本地，缓存配置
		r.createConfigCache(dataId, content)
		// 存入viper
		r.viper.Set(dataId, content)
		// 监听
		err = r.Listen(dataId)
		if err != nil {
			logrus.Warnf("配置监听设置失败: %v", err)
		}
	}
}

// Listen 监听配置变更
func (r *Road) Listen(dataId string) error {
	err := r.nacosClient.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  r.cfg.BaseConfig.Group,
		OnChange: func(namespace, group, dataId, data string) {
			logrus.Info("检测到配置变更，重新加载配置")
			// 重新加载配置
			r.viper.Set(dataId, data)
			// 写入到本地，缓存配置
			r.createConfigCache(dataId, data) // 可以在这里添加配置变更后的处理逻辑
			logrus.Info("配置重新加载成功")
		},
	})
	return err
}

// getDataIds 获取所有dataId
func (r *Road) getDataIds() []string {
	var dataIds []string
	for pageNo := 1; ; pageNo++ {
		param := vo.SearchConfigParam{
			Group:    r.cfg.BaseConfig.Group,
			Search:   r.cfg.BaseConfig.Search,
			PageSize: r.cfg.BaseConfig.PageSize,
			PageNo:   pageNo,
		}
		configs, err := r.nacosClient.SearchConfig(param)
		if err != nil {
			logrus.Fatalf("读取配置文件失败: %v", err)
			panic(err)
		}
		for _, config := range configs.PageItems {
			dataIds = append(dataIds, config.DataId)
		}
		// break
		if pageNo*r.cfg.BaseConfig.PageSize >= configs.TotalCount {
			break
		}
	}
	return dataIds
}

// createConfigCache 创建配置缓存
func (r *Road) createConfigCache(dataId, content string) {
	cacheDir := r.cfg.BaseConfig.CacheDir
	// 判断是否存在，不存在创建
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		if err = os.MkdirAll(cacheDir, 0755); err != nil {
			return
		}
	}
	cacheFile := filepath.Join(cacheDir, dataId)
	if err := os.WriteFile(cacheFile, []byte(content), 0644); err != nil {
		return
	}
}
