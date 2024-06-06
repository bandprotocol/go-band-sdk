package main

import (
	"os"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/viper"
)

type ChainConfig struct {
	ChainID string        `yaml:"chain_id" mapstructure:"chain_id"`
	RPC     string        `yaml:"rpc"      mapstructure:"rpc"`
	Fee     string        `yaml:"fee"      mapstructure:"fee"`
	Timeout time.Duration `yaml:"timeout"  mapstructure:"timeout"`
}

type RequestConfig struct {
	OracleScriptID int    `yaml:"oracle_script_id" mapstructure:"oracle_script_id"`
	Calldata       string `yaml:"calldata"         mapstructure:"calldata"`
	Mnemonic       string `yaml:"mnemonic"         mapstructure:"mnemonic"`
}

type Config struct {
	Chain    ChainConfig   `yaml:"chain"     mapstructure:"chain"`
	Request  RequestConfig `yaml:"request"   mapstructure:"request"`
	LogLevel string        `yaml:"log_level" mapstructure:"log_level"`
	SDK      *sdk.Config
}

func GetConfig(name string) (Config, error) {
	viper.SetConfigType("yaml")
	viper.SetConfigName(name)
	viper.AddConfigPath("./requester/configs")

	if err := viper.ReadInConfig(); err != nil {
		return Config{}, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return Config{}, err
	}

	config.SDK = sdk.GetConfig()

	return config, nil
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
