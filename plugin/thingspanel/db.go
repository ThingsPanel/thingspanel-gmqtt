package thingspanel

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/redis.v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var redisCache *redis.Client
var db *gorm.DB

type Device struct {
	ID             string     `gorm:"column:id;primaryKey;comment:Id" json:"id"`                                                         // Id
	Name           *string    `gorm:"column:name;comment:设备名称" json:"name"`                                                              // 设备名称
	DeviceType     int16      `gorm:"column:device_type;not null;default:1;comment:设备类型（1-直连设备 2-网关设备3-网关子设备）默认直连设备" json:"device_type"` // 设备类型（1-直连设备 2-网关设备3-网关子设备）默认直连设备
	Voucher        string     `gorm:"column:voucher;not null;comment:凭证 默认自动生成" json:"voucher"`                                          // 凭证 默认自动生成
	TenantID       string     `gorm:"column:tenant_id;not null;comment:租户id，外键，删除时阻止" json:"tenant_id"`                                  // 租户id，外键，删除时阻止
	IsEnabled      string     `gorm:"column:is_enabled;not null;comment:启用/禁用 enabled-启用 disabled-禁用 默认禁用，激活后默认启用" json:"is_enabled"`    // 启用/禁用 enabled-启用 disabled-禁用 默认禁用，激活后默认启用
	ActivateFlag   string     `gorm:"column:activate_flag;not null;comment:激活标志inactive-未激活 active-已激活" json:"activate_flag"`            // 激活标志inactive-未激活 active-已激活
	CreatedAt      *time.Time `gorm:"column:created_at;comment:创建时间" json:"created_at"`                                                  // 创建时间
	UpdateAt       *time.Time `gorm:"column:update_at;comment:更新时间" json:"update_at"`                                                    // 更新时间
	DeviceNumber   string     `gorm:"column:device_number;not null;comment:设备编号 没送默认和token一样" json:"device_number"`                      // 设备编号 没送默认和token一样
	ProductID      *string    `gorm:"column:product_id;comment:产品id 外键，删除时阻止" json:"product_id"`                                         // 产品id 外键，删除时阻止
	ParentID       *string    `gorm:"column:parent_id;comment:子设备的网关id" json:"parent_id"`                                                // 子设备的网关id
	Protocol       *string    `gorm:"column:protocol;comment:通讯协议" json:"protocol"`                                                      // 通讯协议
	Lable          *string    `gorm:"column:lable;comment:标签 单标签，英文逗号隔开" json:"lable"`                                                   // 标签 单标签，英文逗号隔开
	Location       *string    `gorm:"column:location;comment:地理位置" json:"location"`                                                      // 地理位置
	SubDeviceAddr  *string    `gorm:"column:sub_device_addr;comment:子设备地址" json:"sub_device_addr"`                                       // 子设备地址
	CurrentVersion *string    `gorm:"column:current_version;comment:当前固件版本" json:"current_version"`                                      // 当前固件版本
	AdditionalInfo *string    `gorm:"column:additional_info;default:{};comment:其他信息 阈值、图片等" json:"additional_info"`                      // 其他信息 阈值、图片等
	ProtocolConfig *string    `gorm:"column:protocol_config;default:{};comment:协议表单配置" json:"protocol_config"`                           // 协议表单配置
	Remark1        *string    `gorm:"column:remark1" json:"remark1"`
	Remark2        *string    `gorm:"column:remark2" json:"remark2"`
	Remark3        *string    `gorm:"column:remark3" json:"remark3"`
	DeviceConfigID *string    `gorm:"column:device_config_id;comment:设备配置id" json:"device_config_id"` // 设备配置id
	BatchNumber    *string    `gorm:"column:batch_number;comment:批次号" json:"batch_number"`            // 批次号
}

func (Device) TableName() string {
	return "devices"
}

// 创建 redis 客户端
func createRedisClient() *redis.Client {
	redisHost := viper.GetString("db.redis.conn")
	dataBase := viper.GetInt("db.redis.db_num")
	password := viper.GetString("db.redis.password")
	log.Println("连接redis...")
	client := redis.NewClient(&redis.Options{
		Addr:         redisHost,
		Password:     password,
		DB:           dataBase,
		ReadTimeout:  2 * time.Minute,
		WriteTimeout: 1 * time.Minute,
		PoolTimeout:  2 * time.Minute,
		IdleTimeout:  10 * time.Minute,
		PoolSize:     1000,
	})

	// 通过 cient.Ping() 来检查是否成功连接到了 redis 服务器
	_, err := client.Ping().Result()
	if err != nil {
		log.Println("连接redis连接失败,", err)
		panic(err)
	} else {
		log.Println("连接redis成完成...")
	}

	return client
}

func createPgClient() *gorm.DB {
	psqladdr := viper.GetString("db.psql.psqladdr")
	psqlport := viper.GetInt("db.psql.psqlport")
	psqluser := viper.GetString("db.psql.psqluser")
	psqlpass := viper.GetString("db.psql.psqlpass")
	psqldb := viper.GetString("db.psql.psqldb")
	connectionString := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=disable", psqluser, psqlpass, psqldb, psqladdr, psqlport)
	// 连接数据库
	log.Println("连接数据库...")
	d, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{})

	if err != nil {
		panic(err)
	} else {
		log.Println("连接数据库成功...")
	}
	return d
}

func Init() {
	redisCache = createRedisClient()
	db = createPgClient()
}

func SetStr(key, value string, time time.Duration) (err error) {
	err = redisCache.Set(key, value, time).Err()
	if err != nil {
		return err
	}
	return err
}

func GetStr(key string) (value string, err error) {
	v, err := redisCache.Get(key).Result()
	if err != nil {
		return "", err
	}
	return v, nil
}

func DelKey(key string) (err error) {
	err = redisCache.Del(key).Err()
	return err
}

// SetNX 尝试获取锁
func SetNX(key, value string, expiration time.Duration) (ok bool, err error) {
	ok, err = redisCache.SetNX(key, value, expiration).Result()
	return
}

// SetNX 释放锁
func DelNX(key string) (err error) {
	err = redisCache.Del(key).Err()
	return
}

// setRedis 将任何类型的对象序列化为 JSON 并存储在 Redis 中
func SetRedisForJsondata(key string, value interface{}, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return redisCache.Set(key, jsonData, expiration).Err()
}

// getRedis 从 Redis 中获取 JSON 并反序列化到指定对象
func GetRedisForJsondata(key string, dest interface{}) error {
	val, err := redisCache.Get(key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

// 通过token从redis中获取设备信息
// 先从redis中获取设备id，如果没有则从数据库中获取设备信息，并将设备信息和token存入redis
func GetDeviceByVoucher(voucher string) (*Device, error) {
	var device Device
	deviceId, _ := GetStr(voucher)
	Log.Debug("缓存的deviceId值: " + deviceId)
	if deviceId == "" {
		result := db.Model(&Device{}).Where("voucher = ?", voucher).First(&device)
		if result.Error != nil {
			Log.Info(result.Error.Error())
			return nil, result.Error
		}
		// 修改token的时候，需要删除旧的token
		// 将token存入redis
		err := SetStr(voucher, device.ID, 0)
		if err != nil {
			return nil, err
		}
		// 将设备信息存入redis
		err = SetRedisForJsondata(deviceId, device, 0)
		if err != nil {
			return nil, err
		}
	} else {
		d, err := GetDeviceById(deviceId)
		if err != nil {
			return nil, err
		}
		device = *d
	}

	return &device, nil
}

// GetDeviceById
// 通过设备id从redis中获取设备信息
// 先从redis中获取设备信息，如果没有则从数据库中获取设备信息，并将设备信息存入redis
func GetDeviceById(deviceId string) (*Device, error) {
	var device Device
	result := db.Model(&Device{}).Where("id = ?", deviceId).First(&device)
	if result.Error != nil {
		return nil, result.Error
	}
	// 将设备信息存入redis
	err := SetRedisForJsondata(deviceId, device, 0)
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetDeviceByNumber fetches device by device_number
func GetDeviceByNumber(deviceNumber string) (*Device, error) {
	var device Device
	result := db.Model(&Device{}).Where("device_number = ?", deviceNumber).First(&device)
	if result.Error != nil {
		return nil, result.Error
	}
	// 缓存一份（使用设备ID作为key）
	_ = SetRedisForJsondata(device.ID, device, 0)
	return &device, nil
}

// 根据token获取订阅信息
type UserPub struct {
	Attribute string `json:"attribute"`
	Event     string `json:"event"`
}
type UserSub struct {
	Attribute string `json:"attribute"`
	Commands  string `json:"commands"`
}
type UserTopic struct {
	UserPub UserPub `json:"user_pub"`
	UserSub UserSub `json:"user_sub"`
}

// func GetUserTopicByToken(token string) (*UserTopic, error) {
// 	var userTopic UserTopic
// 	device, err := GetDeviceByToken(token)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if device.AdditionalInfo == "" {
// 		return nil, fmt.Errorf("empty")
// 	}
// 	// 转map
// 	var additionalInfo map[string]interface{}
// 	err = json.Unmarshal([]byte(device.AdditionalInfo), &additionalInfo)
// 	if err != nil {
// 		return nil, err
// 	}
// 	// 判断有没有pub_topic
// 	if _, ok := additionalInfo["user_topic"]; !ok {
// 		return nil, fmt.Errorf("empty")
// 	}
// 	// additionalInfo["user_topic"]转UserTopic
// 	userTopicJson, err := json.Marshal(additionalInfo["user_topic"])
// 	if err != nil {
// 		return nil, err
// 	}
// 	err = json.Unmarshal(userTopicJson, &userTopic)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &userTopic, nil
// }
