package config

import "github.com/spf13/viper"

type Config struct {
	App   AppConfig `yaml:"app"`
	MySQL DBConfig  `yaml:"mysql"`
	Mongo DBConfig  `yaml:"mongodb"`
}

type DBConfig struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"db_name"`
}

type AppConfig struct {
	PasswordRetentionMinute int    `yaml:"password_retention_minute"`
	Port                    int    `yaml:"port"`
	SecretKey               string `yaml:"secret_key"`
}

var C Config

func LoadConfig(path string) error {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	C.App.PasswordRetentionMinute = viper.GetStringMap("app")["password_retention_minute"].(int)
	C.App.Port = viper.GetStringMap("app")["port"].(int)

	C.MySQL.Port = viper.GetStringMap("mysql")["port"].(int)
	C.MySQL.User = viper.GetStringMap("mysql")["user"].(string)
	C.MySQL.Password = viper.GetStringMap("mysql")["password"].(string)
	C.MySQL.Host = viper.GetStringMap("mysql")["host"].(string)
	C.MySQL.Name = viper.GetStringMap("mysql")["db_name"].(string)

	C.Mongo.Host = viper.GetStringMap("mongodb")["host"].(string)
	C.Mongo.Name = viper.GetStringMap("mongodb")["db_name"].(string)
	C.Mongo.Port = viper.GetStringMap("mongodb")["port"].(int)

	return nil
}
