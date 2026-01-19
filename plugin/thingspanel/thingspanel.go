package thingspanel

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/DrmagicE/gmqtt/config"
	"github.com/DrmagicE/gmqtt/server"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var _ server.Plugin = (*Thingspanel)(nil)

const Name = "thingspanel"

var (
	runtimeInitOnce sync.Once
	runtimeInitErr  error
	Log             *zap.Logger
)

func init() {
	server.RegisterPlugin(Name, New)
	config.RegisterDefaultPluginConfig(Name, &DefaultConfig)
}

func runtimeInit() error {
	log.Println("thingspanel: initializing config...")
	viper.SetEnvPrefix("GMQTT")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetConfigName("thingspanel")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("thingspanel: failed to read configuration file: %w", err)
	}

	Init() // init database & redis
	go DefaultMqttClient.MqttInit()
	return nil
}

func New(config config.Config) (server.Plugin, error) {
	return &Thingspanel{}, nil
}

type Thingspanel struct{}

func (t *Thingspanel) Load(service server.Server) error {
	Log = server.LoggerWithField(zap.String("plugin", Name))
	runtimeInitOnce.Do(func() {
		runtimeInitErr = runtimeInit()
	})
	return runtimeInitErr
}

func (t *Thingspanel) Unload() error { return nil }

func (t *Thingspanel) Name() string { return Name }

// Deprecated: not used.
func (t *Thingspanel) UpdateStatus(accessToken string, status string) {
	url := "/api/device/status"
	method := "POST"
	payload := strings.NewReader(`"accessToken": "` + accessToken + `","values":{"status": "` + status + `"}}`)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}
